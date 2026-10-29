# terminu-go

**Language**: [English](README.md) · 繁體中文

**照 [terminu design](https://github.com/vulcanshen/terminu) 畫出並驅動終端機介面的 Go 函式庫。**

你登記 panel 與動作 —— 鍵、名稱、範圍（item、panel 或 global）、執行前要不要先問、做什麼。terminu-go 照這一份登記畫出
整個畫面，並處理 core key：`Space` menu、`?` 的 key reference、`!` 的 App Log（每一個 toast 與錯誤，加上 app 記的事）、
每個 panel 下框的 hint、footer、confirm、淡化沒有 focus 的地方、`Esc` 一次關一層、popup 上再疊 popup。menu、key reference 與實際的按鍵出自同一份登記，所以不會對不上。

函式庫管不到的畫面（全螢幕的鎖定畫面、好幾個並排的終端機），照樣可以用它的零件：用 `App.Canvas` 自己畫那個畫面，
confirm、toast 與每一種 input 照樣疊在上面，跟在其他地方一樣。
app 自己的 popup（例如畫格子的）也放進同一疊：框裡畫什麼、按鍵做什麼由你決定；外框、開關動畫、淡化與 `Esc` 交給函式庫，
函式庫的 confirm 與 input 也會疊在你的 popup 上面。裡面有要送出的值時，它就照表單：按鈕由函式庫放在你的內容下面，`Tab`、
錯誤列，以及 `Esc` 丟掉改過的內容前先問，也由函式庫做。

建在 [Bubble Tea](https://github.com/charmbracelet/bubbletea) v1 與 [Lip Gloss](https://github.com/charmbracelet/lipgloss) v1 上。

## 安裝

```sh
go get github.com/vulcanshen/terminu-go
```

需要 Go 1.26 以上。用它做的 app，對使用者終端機的需求跟 terminu design 一樣：[Nerd Font](https://www.nerdfonts.com/) 與
24-bit 色（truecolor）。終端機元件（在 PTY 裡跑程式）只能在 Unix 上用。

有些字型讓 icon 佔兩格，有些連 powerline 符號（框上膠囊的兩端、cursor 尾端的箭頭）也佔兩格。函式庫在 app 啟動時兩種都自己量；
使用者也可以直接指定：`<APP>__ICON_WIDTH` 與 `<APP>__POWERLINE_WIDTH`（只給一個 app，例：`LOCKU__ICON_WIDTH=2`），或
`TERMINU__ICON_WIDTH` 與 `TERMINU__POWERLINE_WIDTH`（terminu 家族共用：設一次，家族每個 app 都讀得到）。在 app 裡面跑終端機的 app，
會替它開的程式把兩個 `TERMINU__` 變數都設好。

## 快速開始

一個完整的 app：一份待辦清單，上面一個動作（[`examples/minimal`](examples/minimal/main.go)）：

```go
// minimal is the smallest terminu app: one list and one action on it.
package main

import (
	_ "embed"
	"log"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

//go:embed icon.svg
var icon []byte // the app's mark, drawn by its splash (V)

func main() {
	app := terminu.New("todo")
	list := &terminu.List{Columns: []terminu.Column{{Title: "Task", Shrink: true}}}
	list.SetRows([]terminu.Row{{ID: "1", Cells: []string{"Write the README"}}, {ID: "2", Cells: []string{"Tag v1.0.0"}}})
	done := terminu.Action{Key: "x", Label: "Done", Desc: "this task", Scope: terminu.ItemScope, Hint: "done",
		Do: func() tea.Cmd {
			row, _ := list.Current()
			list.SetRows(slices.DeleteFunc(list.Rows(), func(r terminu.Row) bool { return r.ID == row.ID }))
			return app.Toast(terminu.Info, "Done: "+row.Cells[0])
		}}
	app.AddScreen(&terminu.Screen{Name: "Tasks", Panels: []*terminu.Panel{{Label: "Tasks", Body: list, Actions: []terminu.Action{done}}}})
	var err error
	if app.Splash, err = terminu.NewSplash(icon, "todo", "v0.1.0", "Things to do"); err != nil {
		log.Fatal(err)
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`x` 把 cursor 所在的待辦標成完成。`Space` 列出這裡能做什麼，`?` 列出按鍵，`V` 打開 splash，`q` 離開。每個 terminu app 都有
一個畫著自己 mark 的 splash（tdp S1、S5）；`icon.svg` 就是那個 mark，這個範例用的是 terminu 的副本。

想看它做得到的全部，跑展示場（用假資料展示每一個元件的範例 app）：

```sh
go run github.com/vulcanshen/terminu-go/examples/showcase@latest
```

## 文字與顏色

交給函式庫的字串都是文字。從外面來的東西（檔名、網頁標題、程式的輸出）一律不原樣送給終端機：

- **名字與短字**（清單的格子、表單與 panel 上的值、menu 的 label、標題、toast、confirm、選項、statusbar）：每個控制字元都畫成
  紅色的名字，Tab 是 `\t`，顏色碼是 `\x1b` 接著 `[31m`。要自己上色時另外給 span：`Row.Styled`、`Field.Styled`、
  `FormField.Styled`，`App.Status` 也回 span。在 `Canvas` 或自己的 popup 裡畫外面來的字時，用 `Escape` 照同一套畫。
  清單裡放一行一行文字的欄（原始碼的一行、log 的一行）標成 `Column.Line`：裡面的 Tab 畫成空白。
  自己寫的字不放換行字元：好幾列的訊息一列一列給，`app.Error(title, row1, row2)`、`app.Toast(kind, row1, row2)`，
  畫面上紅色的 `\n` 就一定是外面來的。
- **拿來讀的內容**（viewer、panel 上的 log、預覽）：照終端機會畫的樣子，顏色碼照著上色、`\n` 換行、進度列的 `\r` 只留最後的
  狀態，其他畫成紅色的名字。自己畫的 body 要顯示 log 時，用 `Reading` 畫它的每一行。

## App Log

在任何 panel 上按 `!` 打開 App Log：app 做過的事與出問題的事，最新的在最下面，只放在記憶體裡。每一個 toast 與 error popup
都會自己記進去；其他的用 `app.Log(kind, text)` 記（動作的結果、app 在背景做的事），任何 goroutine 都能叫；寫法跟 toast
一樣，大寫開頭。有沒看過的錯誤或警告時，footer 的 `!:log` 會寫出有幾個。

## 讀不了自己的檔時

app 讀不了自己的設定、狀態或資料（格式錯、檔案壞掉、沒有權限）時，在啟動時叫 `app.MoveAside(path, err, startFresh)`。
使用者會被問要不要把檔移開：`Enter` 改名成 `.bak`（一律不蓋掉別的）再跑 `startFresh`，app 在裡面像第一次打開一樣開始；
`Esc` 離開，`Run` 回傳路徑與原因，交給 `main` 印出來。資料是一筆一個檔時，讀不了的那一筆改叫
`app.MoveAsideRecord(path, err, reread)`：`Enter` 只把那一筆移開，再跑 `reread` 讀其他的。改版後不再用的檔交給使用者處理：
每次啟動叫 `app.NoLongerUsed(paths...)`，還在的那幾個會在 App Log 各記一筆。

## 要提權時

要以管理員身分做事（例：把暫存檔複製到 `/etc/hosts`）時，叫 `app.Elevate(terminu.Elevate{Command: ..., Done: ...})`，只給要跑的
指令。不用密碼時直接跑；要密碼時 lib 開 `PASSWORD` 的 password popup，把密碼交給 sudo，打錯了留在 popup 裡重打。app 拿不到密碼：
它不會被存下來、記進 App Log、出現在畫面上或送出去。`Done` 拿到指令怎麼結束的：失敗時是 `*terminu.ElevateError`
（指令印出來的每一列），交給 error popup；使用者取消時跑 `OnCancel`。沒有 sudo 只有 doas 時設 `Tool: terminu.Doas`，doas
在 terminal popup 裡自己問密碼。Windows 不用這個，照舊用 gsudo 或 Windows 的 sudo（系統跳 UAC）。

## 設定

使用者的設定裡，有兩個是 terminu 家族每個 app 共用的（tdp E3）。函式庫不讀設定檔：由 app 從自己的設定檔讀出來交給它。

| key | 作用 | 預設 | 交給 |
|---|---|---|---|
| `hide_secrets_in_preview` | 檔案預覽裡藏起私鑰與只有擁有者能讀的檔 | `false` | `App.HideSecretsInPreview` |
| `input_tab_width` | 多行文字裡按 `Tab` 插入幾個空白；畫面上文字裡的 Tab 也畫成這麼寬 | `4` | `App.InputTabWidth` |

## 測試 app

`tdptest` 套件在 Go 的測試裡像使用者那樣操作 app，每處理一個 message 就檢查 terminu design 對每個畫面的要求：每一列剛好等於
終端機的寬度（tdp L4）。

```go
func TestDone(t *testing.T) {
	d := tdptest.Start(t, newApp(), 80, 40) // newApp 是 main 裡 Run 以外的全部
	d.Press("x")
	if !d.Has("Done: Write the README") {
		t.Error("no toast")
	}
	d.Advance(10 * time.Second)
	if d.Has("Done:") {
		t.Error("the toast stays")
	}
}

// 第一次啟動，在好幾種終端機尺寸上走一遍。
func TestFirstLaunch(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "todo", New: newApp})
}
```

- **按鍵與文字**：`Press` 用 terminu design 寫的鍵名（`"j"`、`"Enter"`、`"Ctrl-C"`、`"Alt-Esc"`）；`Type` 一個字一個鍵打字，
  `Burst` 一次送進一串，`Paste` 貼上。`Resize` 改終端機大小，`Send` 送 app 自己的 message，`Do` 在可以改 app 的地方跑程式
  （`app.Focus`）。
- **時間由測試決定**：每次輸入之後，popup 的動畫播到完；`Advance` 讓時間往前走（toast 消失、計時器到時）；設 `d.Auto = false`
  時用 `Step` 一格一格播。app 自己的計時器要用 `App.Tick` 做，才跟著測試的時間走（見下面）。
- **命令**照程式裡那樣跑。跑得久的（讀檔、程式的輸出）在之後的輸入時回來；`WaitFor` 等它。
- **畫面上有什麼**：`Screen` 是每一列的文字，`Styled` 帶顏色，`Has` 找字，`Quit` 回報 app 要離開了沒，`Clipboard` 是它複製了什麼。
  失敗時會寫出尺寸、按過的鍵與整個畫面；`Note` 再多寫一句（例：給 app 的是 `Tricky` 的哪一個）。
- **`EachIconWidth`** 把一個測試跑三次：icon 與 powerline 符號都佔一格、icon 佔兩格、兩種都佔兩格。
- **`Check`** 在好幾種終端機尺寸、每一種 icon 寬度下，像第一次那樣啟動 app，每次都在新的沙盒裡（`HOME` 與 XDG 目錄放在 `/tmp`
  底下，app 自己的環境變數清掉）；在每個畫面、每個 panel 打開 `Space` menu、global operation popup、App Log 與 `?`，再用 `Esc` 一層層關掉；
  最後把終端機縮到比 app 的最低尺寸小，再放回來。`MinWidth`、`MinHeight` 高過 60×32，或 `NarrowBelow` 高過 60，也會失敗（tdp L1）。
  畫面上出現被畫成字的顏色碼（留在格子或值裡、沒改成 `Styled` 的）也會失敗；
  `Tricky` 標明 Seed 放進去的是刁鑽值時不算。任何用 `Driver` 的測試，畫面上出現紅色的換行名字（自己的 toast 裡寫了 `\n`）都會
  失敗；資料刻意放了這種名字時設 `AllowLineBreaks`。
  `Env` 拿得到沙盒的 `HOME`，環境變數可以指到 `Seed` 放進去的檔。
  `go test -short` 時只在最低尺寸與 80×24 啟動，icon 只用一格。
  它先在子測試 `repo` 裡讀 app 的 repo（測試目錄往上找到的 `go.mod`），照 tdp project 檢查：範本（`family.mk`、`release.yml`、
  `.goreleaser.yaml`、`install.sh`、`uninstall.sh`、`version.go`、`.gitignore`）要跟這一版實作的 tdp 一樣，只差每一份允許 app 改的地方；
  每個 app 都有的檔案；Makefile 的排法；CHANGELOG 的開頭與標題；git 沒有追蹤 `CLAUDE.md`，`.claude/` 底下只追蹤 harness 的
  `.claude/rules/terminu.md`。失敗時寫出哪個檔、差在哪裡、要重新複製哪一份範本。
  tdp project 只綁家族成員：`go.mod` 的 module 不在 `github.com/vulcanshen` 底下的 app，repo 不查。
- **`Tricky`、`Sizes` 與 `Mashes`** 是弄壞過畫面的值：控制字元、escape sequence、寬字與組合字、很長的值；奇怪的終端機尺寸；
  比動畫還快的連按（`Mash`）。

icon 的寬度是整個測試程式共用的，所以操作 app 的測試不跟別的測試平行跑。

### 計時器：`App.Tick`

app 自己的計時器用 `App.Tick` 做，不用 `tea.Tick`；參數一樣：

```go
return app.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return searchNow{} }) // 原本是 tea.Tick
```

在程式裡它就是 `tea.Tick`。在測試裡它等測試的時間，`Advance` 推到了才觸發，不會自己跑。用 `tea.Tick` 做的計時器在測試裡也照
真的時間跑：測試推不動它，它做的事在時間到的時候才進來。

## 實作哪一版 terminu design

terminu-go 實作 **tdp v1.1.1**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
