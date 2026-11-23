//go:build windows

package terminu

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// conpty is the PTY Windows has, ConPTY: Windows 10 1809 and later.
var conpty = windows.NewLazySystemDLL("kernel32.dll").NewProc("CreatePseudoConsole")

type winProcess struct {
	proc *os.Process
	in   *os.File // the program's input: keys go here
	out  *os.File // what it prints
	// mu keeps the update loop resizing the pseudo console apart from
	// closing it; pc is 0 once closed.
	mu   sync.Mutex
	pc   windows.Handle
	done chan struct{} // closed once the program has ended
	err  error
}

func (p *winProcess) Read(b []byte) (int, error)  { return p.out.Read(b) }
func (p *winProcess) Write(b []byte) (int, error) { return p.in.Write(b) }

func startPTY(argv []string, dir string, env []string, w, h int) (io.ReadWriter, process, error) {
	if conpty.Find() != nil {
		return nil, nil, errors.New("a terminal needs Windows 10 1809 or later")
	}
	path, err := programPath(argv[0], dir)
	if err != nil {
		return nil, nil, err
	}
	var inR, inW, outR, outW windows.Handle
	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return nil, nil, err
	}
	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		closeHandles(inR, inW)
		return nil, nil, err
	}
	var pc windows.Handle
	err = windows.CreatePseudoConsole(windows.Coord{X: int16(w), Y: int16(h)}, inR, outW, 0, &pc)
	// The console keeps copies of its ends: with ours closed, what it prints
	// ends once it is closed.
	closeHandles(inR, outW)
	if err != nil {
		closeHandles(inW, outR)
		return nil, nil, err
	}
	proc, err := spawn(path, argv, dir, env, pc)
	if err != nil {
		closeHandles(inW, outR) // first: closing waits for what it printed to be read
		go windows.ClosePseudoConsole(pc)
		return nil, nil, err
	}
	p := &winProcess{proc: proc, pc: pc, done: make(chan struct{}),
		in: os.NewFile(uintptr(inW), "conpty-in"), out: os.NewFile(uintptr(outR), "conpty-out")}
	go func() {
		state, err := proc.Wait()
		if err == nil && !state.Success() {
			err = &exec.ExitError{ProcessState: state}
		}
		p.err = err
		// The program has ended but the console stays until closed, and
		// what it printed ends only then.
		p.close()
		close(p.done)
	}()
	return p, p, nil
}

// programPath finds the program as os/exec does: a name on the PATH; a path
// as it is, a relative one from dir; on both an extension from PATHEXT when it
// has none.
func programPath(name, dir string) (string, error) {
	if dir != "" && strings.ContainsAny(name, `\/`) && filepath.VolumeName(name) == "" && !os.IsPathSeparator(name[0]) {
		name = filepath.Join(dir, name)
	}
	return exec.LookPath(name)
}

// spawn starts the program at path, with argv, in the pseudo console pc.
func spawn(path string, argv []string, wd string, env []string, pc windows.Handle) (*os.Process, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, err
	}
	defer attrs.Delete()
	// The attribute's value is the handle itself, not where it is kept.
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&pc)), unsafe.Sizeof(pc)); err != nil {
		return nil, err
	}
	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	// None of the app's standard handles: redirected (as under go test), the
	// program would write there instead of in the console.
	si.Flags = windows.STARTF_USESTDHANDLES
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	line, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(argv))
	if err != nil {
		return nil, err
	}
	var dir *uint16
	if wd != "" {
		if dir, err = windows.UTF16PtrFromString(wd); err != nil {
			return nil, err
		}
	}
	// Environ drops a variable given twice but the last, Windows' names in
	// any case, and adds SYSTEMROOT, as os/exec does.
	block, err := envBlock((&exec.Cmd{Env: env}).Environ())
	if err != nil {
		return nil, err
	}
	var pi windows.ProcessInformation
	err = windows.CreateProcess(name, line, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT, &block[0], dir, &si.StartupInfo, &pi)
	if err != nil {
		return nil, err
	}
	// The handle held, the program's id cannot go to another before
	// FindProcess opens its own.
	defer closeHandles(pi.Process, pi.Thread)
	return os.FindProcess(int(pi.ProcessId))
}

// envBlock is env as CreateProcess takes it: each "key=value" ended by a NUL,
// one more NUL after the last.
func envBlock(env []string) ([]uint16, error) {
	var b []uint16
	for _, kv := range env {
		s, err := windows.UTF16FromString(kv) // ends with its NUL; refuses one inside
		if err != nil {
			return nil, err
		}
		b = append(b, s...)
	}
	if len(b) == 0 {
		b = append(b, 0)
	}
	return append(b, 0), nil
}

func closeHandles(hs ...windows.Handle) {
	for _, h := range hs {
		_ = windows.CloseHandle(h)
	}
}

// close closes the pseudo console once; it waits for what the program
// printed to be read, so it never runs on the update loop.
func (p *winProcess) close() {
	p.mu.Lock()
	pc := p.pc
	p.pc = 0
	p.mu.Unlock()
	if pc != 0 {
		windows.ClosePseudoConsole(pc)
	}
}

func (p *winProcess) resize(w, h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pc != 0 {
		_ = windows.ResizePseudoConsole(p.pc, windows.Coord{X: int16(w), Y: int16(h)})
	}
}

// stop closes the console, as closing a console window does: the programs
// in it are told to close (CTRL_CLOSE_EVENT) and end.
func (p *winProcess) stop() { go p.close() }

// wait comes once reading has failed: the output is closed first, so that
// a console still open after some other failure is not left waiting for it
// to be read.
func (p *winProcess) wait() error {
	p.out.Close()
	<-p.done
	p.in.Close()
	return p.err
}
