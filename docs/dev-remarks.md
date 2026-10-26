# terminu-go 開發者備忘

terminu-go 是 terminu design principle（tdp）的 Go 實作：給 app 呼叫的 lib，加上展示場 `examples/showcase`。遵循 terminu design
principle，目前對應 tdp v0.10.1。這個 repo 的任務與規範的讀法見 [`CLAUDE.md`](../CLAUDE.md)。

## 運作方式

lib 建在 bubbletea v1（v1.3.10）與 lipgloss v1（v1.1.0）上，跟家族的五個 app 一樣。

- **寬度**：所有量寬度的地方走同一個函式（tdp E5）：補空白、截斷、置中、並排、框線、疊 popup。Nerd Font 的 icon（PUA）照探測到的
  格數算，其他照 `x/ansi`。lip gloss 只拿來上色（`Foreground`、`Background`、`Bold` 套在量好寬度的字串上）；`Style` 的
  `Width`、`Padding`、`Border`、`Align`，`JoinHorizontal`／`JoinVertical`、`Place`，以及 bubbletea-overlay 都不用 —— 它們用自己的
  寬度，icon 一律算一格。
- **畫到終端機**：用 bubbletea v1 的 renderer。它把改變的列從第 0 欄整列重寫，終端機自己決定每個字落在哪一格，所以 icon 佔兩格
  也對得上（`proto/c1` 在 icon 兩格的 tmux 裡逐字比對過，見盲點紀錄 C1 的「v1 重驗」）。
  - 例外是 renderer 在列尾補的「清到行尾」：它用 `ansi.StringWidth`（icon 算一格）量每一列，比終端機寬短的列後面補 `ESC [ K`。
    icon 兩格時，含 icon 的列量起來一定比較短，整列寫滿之後還會送一次 `ESC [ K`，照 xterm 規矩的終端機（Ghostty 也是）會把最後一格
    （右框）擦掉；tmux 不會。lib 畫的每一列都剛好是終端機寬（L4），這個清除永遠是多餘的，所以 `App.Run` 交給 bubbletea 的輸出
    （`output.go` 的 `screenOutput`）把它拿掉。

- **App 是一個 bubbletea model**：app 登記畫面（`Screen`）、panel（`Panel`，內容是 `Body`：`List`、`Values`、`Pager` 或 app 自己的）、
  動作（`Action`：鍵、名稱、說明、範圍、要不要先問、做什麼）。Space menu、global operation popup、`?` 的 key reference、panel 下框的 hint、
  footer 都從這一份登記產生（`app.go`、`menu.go`、`note.go`）。
- **一個畫面怎麼畫出來**（`App.Frame`）：最上面一列（畫面 chip 列加分隔線，或 statusbar）、panel 區（每個 panel 畫好框再用 `Composite`
  放上去；失焦的 panel 內容淡化一次，串流除外）、footer。有 popup 時整個底下淡化一次，再由下往上疊每一層 popup（最上層以外都淡化），
  最後疊 toast。每一層都照開關動畫的那一格畫（`layer.drawFrame`）。
- **按鍵**（`handleKey`）：開到一半的 popup 不收鍵 → 離開流程裡的第二個 `Ctrl-C` → toast 先收 `Esc` → `Ctrl-C` 進離開流程 → 最上層
  popup → focus 的 panel（輸入態、模式、一般）。`g` 開頭的兩鍵組合（`gg`、`go`）在 app 這一層接起來，body 收到的是 `gg`。
- **input**：每一種 input 都是一個 `Input`（`TextInput`、`Select`、`DatePicker`…），單獨打開用 `App.Open`（或 `App.Text` 這類捷徑）；
  表單的欄位用 `Edit` 在打開時做出它的 `Input`，表單打開它時傳一個 `written`，值寫回時表單記成「改過了」（離開前先問）。一般的
  finder、select、checkbox popup、file-picker 共用 `finder.go` 的骨架（打字列、筆數、分隔線、清單、預覽），各自只換標記、`Enter`
  與清單的來源。
- **上色**：自己寫 SGR（`Style.Render`，一律 24-bit），淡化（`Dim`）就是改寫每一列的 SGR。淡化往 `#0c0c13`（`dimTo`）靠，
  不是 Base：比 Base 暗的顏色也跟著變暗，膠囊串沒選中的段的 Mantle 被蓋住時就成了 Crust，不用另外寫特例。
- **膠囊串**（`drawChain`）：串裡每一格都寫明前景與背景（沒選中的段與接縫是串色配 Mantle），只有兩端外側不塗，所以終端機的底
  是什麼顏色都一樣（kbu 的 KB26）。`[N]` 與第一個 tab 由 `Panel.capsules` 併成一段，中間的空白一個 powerline 符號寬；`drawChain`
  遇到兩段相鄰都選中時也畫同樣的空白。
- **app 自己的 popup**（`apppopup.go`）：`App.Popup` 把 app 的 `PopupBody` 包成 popup 疊裡的一層，跟 lib 自己的 popup 一樣走
  `measure`、`draw`、`key`。app 給框裡的大小（`Size`）、內容（`Draw`，框線到框線的寬度）、按鍵（`Key`）、hint、key reference 的鍵；
  框、標題、層色、動畫、淡化，以及 app 沒收的 `Esc`、`?`、`q` 都是 lib 的。`ClosePopup` 用 body 找到那一層，跟 `closeView` 一樣連上面
  的一起關。body 可以多一個 `QuitNote() string`：離開流程開 confirm 時從下往上問每一個開著的 app popup，不是空的排在 Details 最後、標 Peach。

- **tdptest**（`tdptest/`）：在 Go 的測試裡跑一個 App，不開 bubbletea program，直接叫 `Init`、`Update`、`Frame`，順序照 program：
  `Init`、還不知道大小時畫一次、`WindowSizeMsg`。每個 `Update` 之後畫一次，量每一列（L4：列數、寬度、沒有 SGR 以外的控制字元）。
  - 命令：app 的命令各自在 goroutine 跑，送進去之前照開始的順序排好；這次輸入開始的命令最多等 `Driver.Wait`（200ms），沒回來的
    留到之後的輸入或 `WaitFor`。`tea.BatchMsg` 展開；`tea.Sequence` 的 message 型別沒有匯出，用 reflect 認（元素是 `tea.Cmd` 的
    slice），一個跑完才跑下一個；`tea.QuitMsg` 記下來，之後的輸入讓測試失敗。
  - 時間：`Start` 把 App 放上 driver 的時鐘（`internal/testhook.UseClock`）。lib 的計時器（`App.after`：開關動畫、loading icon、toast、
    splash、輸入框的暫停）與 `App.Tick` 在測試的時鐘上不等待，回一個 `testhook.Timer`，driver 收進自己的排程，`Advance`、`Step`、
    播完動畫時照時間順序觸發。loading icon 的那一格由畫的那一刻的時鐘決定（`spinAt`，`Frame` 開始時設）。
  - `Check`：每種尺寸、每種 icon 寬度都重新 `New` 一次（第一次啟動），沙盒在 `/tmp/tdp-*`；畫面、panel 用 `testhook.Screens` 列出，
    用 `ShowScreen`、`Focus` 切過去，popup 用按鍵開（`Space`、`Space G Enter`、`?`），用 `testhook.Layers` 數層數。

- **控制字元**（`text.go`）：app 給的字在畫的那一刻才處理，資料本身不動（清單照原字篩選、排序）。
  - 名字與短字走 `Escape`：每個控制字元換成紅色的名字，前面是 lib 自己的標記 `nameSGR`（Red 加一個不改變什麼的 `24`）。
    `Style.Render` 在裡面的 reset 之後補回外層的樣式；帶底色的樣式（cursor、膠囊）與 `onBlock` 認得這個標記，把名字畫成
    Red 底、Base 字。app 要上色的地方給 `Span`，lib 把每段的字 `Escape` 之後套上它的樣式。
  - 拿來讀的內容先過 `readingRows`（在 `\n` 處拆行、CRLF 的 `\r` 留在行尾給複製用、`lastState` 處理行裡的 `\r`），再照既有的
    viewer 畫法（`visibleText`、`redNames`：SGR 照留、其他畫名字）。viewer 的 `sync` 在 Lines 變了時重算，`Pager` 在
    `SetLines`、`Append` 時算。
- **App Log**（`applog.go`）：`App.Log` 加鎖，任何 goroutine 都能叫；`Toast`、`ErrorAbout`、`commitFailed`、表單動作的失敗都會
  記一筆。打開時把現在的內容放進一個 viewer，`Frame` 開始時（`syncLog`）把新的一筆補進去，原本在最下面就跟著最下面；它關掉
  之後新的錯誤、警告才算「沒看過」。footer 的說明直接帶顏色碼（`Fg(Red).Render(...)`），`Style.Render` 會在後面補回 Overlay0。
- **互動 terminal**：`Terminal.Interactive` 才有 History 與選取模式（`Alt-v` 寫死在 lib，不是 app 挑的）；子程序在 alternate
  screen 時（`vt.altOn`）`PgUp` 給它；`AllKeys` 時只有出口鍵與 app 的 `Keys` 留在 app。
- **讀不了時**（`moveaside.go`）：confirm 多一個 `quit`，`Esc` 不取消而是記下 `exitErr`、回 `tea.Quit`；`Run` 在程式結束後
  回傳它，`main` 照它處理 `Run` 的錯誤那樣印到 stderr。一筆一個檔的資料用 `MoveAsideRecord`：同一個 confirm，只有問句的
  後半不同；改名失敗時再問的也是同一種問句。lib 只改名，不建新檔：第一次打開會建什麼只有 app 知道（tdp v0.9.2 的「像第一次
  打開一樣」），交給 `fresh`。
- **lib 自己送出自己的計時**（`own`、`sendTimers`）：開關動畫的 tick 同一時間只排一個（`a.anim`），toast 有自己的時間
  （`toastDone`）。這些計時原本只放在 App 方法回傳的 command 裡，app 只回傳兩個裡的一個（同一個 update 開兩個 toast）或全丟掉，
  `a.anim` 就永遠是 true、之後所有開關動畫都停（filu 踩到）。現在 lib 把它們記下來，在 `Update`、`Init`、tdptest 的 `Do`
  結束時自己也送一份；App 方法照樣回傳同一個 command（`ClosePopup` 還是用 nil 說「沒開著」），兩份共用一個 `sync.Once`，
  先跑的那份才排計時，另一份什麼都不給，動畫不會變兩倍快。tdptest 認得這種 command，照樣當場跑（`testhook.Once`）。
- **一行文字的欄**（`Column.Line`）：格子的字（含 `Styled` 每一段的字）先把 Tab 換成 `input_tab_width` 格空白再 `Escape`，
  其他控制字元照樣畫名字；app 給的 span 不改，換在複製的那份上。

`proto/c1` 是驗證盲點紀錄 C1 的拋棄式原型，不是 lib 的一部分；lib 寫好後刪掉（go.mod 裡的 lipgloss 只有它在用）。

## 設計決定

- **bubbletea v1，不是 v2**（盲點紀錄 C1、G22）：bubbletea v2 的 renderer 把 `View` 拆成自己的格子、只補改變的格子，lipgloss v2 的
  排版與 `Compositor`／`Canvas` 也是；兩者的寬度都換不了，icon 佔兩格時框線歪掉、更新後消失。要用 v2 就得自己重寫輸出層。
  user：v2 還不成熟，不為它打破既有狀態、重新造輪；等 v2 的 renderer 與排版能指定寬度函式（或正確處理 icon 佔兩格）時再升。
  輸出那一層盡量寫成換得掉的。
- **寬度與疊 popup 自己做**：理由見上面「運作方式」。
- **PTY 的終端模擬器自己寫**（`vt.go`）：PTY 裡的程式被告知 icon 佔幾格（`TERMINU__ICON_WIDTH`），照它排版；模擬器放字的寬度要跟 lib
  的 `Width` 一樣，畫出來的每一列才會剛好是框裡的寬度。`charmbracelet/x/vt` 的寬度寫死成 `ansi.GraphemeWidth`（icon 一格）、換不了，
  而且帶進 bubbletea v2 那一套（ultraviolet、x/ansi v0.11）；`hinshun/vt10x` 每個字都當一格，連中文都不對。開 PTY 用
  `creack/pty`（MIT，有 tag）。模擬器會回答家族 app 啟動時問的事（CPR、DA1、OSC 4 調色盤），在 PTY 裡跑另一個家族 app 也不用等探測逾時。
- **連上色也不用 lip gloss**：lip gloss v1 的顏色要先設全域的 color profile（`lipgloss.SetColorProfile`），lib 去改它會改到用 lib 的 app；
  `Style.Render` 碰到多列字串還會用自己的寬度補空白。lib 要的只有「這段字用這個顏色」，自己寫 SGR 就夠了，輸出也一定是 24-bit
  （淡化的前提）。app 內容的顏色要用 lip gloss 照用，淡化照樣改得到它的 SGR。
- **app 自己的 popup 進 lib 的疊**（2026-10-15，user）：kamonu 畫格子的 popup 原本插不進 lib 的疊，kamonu 只好不用 `App.Run`、
  自己包一層 program，照抄 Probe 與拿掉 `ESC [ K`，連疊在上面的 confirm、文字輸入、error popup、key reference 都自己畫。kamonu 記的
  KM2（「lib 不放 app 自己的 popup」）理解成 lib 不做格子這種元件；通用的入口可以做：內容與按鍵歸 app，框與 core key 歸 lib。
  在五個 app 改用 lib 之前把 API 定下來，現在改只動到 kamonu 與 ccdrawu。
  - `Draw` 拿框線到框線的寬度，左右的留白 app 自己留：layout/popup 規定整列 cursor 與表單的 focus 色塊蓋住留白，lib 先留好就做不到。
  - 離開的 confirm 多一句用 `QuitNote`，不讓 app 自己換 `App.Quit`：kamonu 的回饋，沒存的狀態在三處變、三處復原，漏一處主畫面的
    `Quit` 也會帶著那句；在離開流程開始時問 body，不用存狀態。`App.Quit` 是 nil 時有 note 照樣先問（`Quit <app>?`）：app 給了 note
    就是要使用者看到。
  - 表單送出成功只關表單自己（`closeOnly`），不是關最上層：kamonu 在 `NEW KAMON` 的 Submit 裡開 `DRAW`，關最上層會關掉剛開的
    `DRAW`、留下表單。層色也改成只數底下開著的層，`DRAW` 從第一格就是第 1 層色，不會等表單收完才跳色。在收的那層不算一層，規範原本沒寫，G90 補進 tdp v0.6.10。
  - 打字列、模式、loading、下框的捲動位置先不做；要的時候用 optional method 加，`PopupBody` 不變。
- **icon 與 powerline 符號各量各的**（tdp E5）：兩種寬度各自照「app 的變數 → 家族的變數 → 隔著 ssh 帶過來的 → 探測」決定
  （`<APP>__ICON_WIDTH`／`TERMINU__ICON_WIDTH`／`LC_TERMINU_ICON_WIDTH`，powerline 符號同一套）；探測時印一個 file icon、
  一個 powerline 符號，各問一次游標位置。有整列 cursor 的地方，右邊的留白跟箭頭一樣寬（layout/popup、layout/panel）。
- **從 menu 跑的動作預設保留 menu**（tdp v0.8.0 的 T1）：lib 分不出動作是哪一種，只看它有沒有打開 popup —— 打開了就把 menu
  留在底下（T1 的預設），什麼都沒打開就關掉 menu（它的事做完了）；清掉 source 要 app 標 `Action.Clear`。confirm 只是門，接受之後
  照它後面的動作判。v0.1.13 以前預設是清掉，改用 lib 的 app 要替開 shell、換畫面、drill-down 的動作補 `Clear`。
- **檔案預覽在背景讀**（tdp v0.8.2）：finder 與 file-picker 的預覽、metadata 在畫的時候只登記要讀什麼（照路徑與框的大小），
  下一次 update 才用 tea.Cmd 在背景讀，讀到了存起來；讀的時候 loading icon 一直轉，frame 會一直來，登記的東西不會等到下一個按鍵。
  panel 的 `FilePreview` 還是畫的時候讀。
- **剪貼簿走終端機**（OSC 52）：viewer 預設的複製把字交給 `Clipboard`，跟著下一個畫面在同一次 write 送出（`screenOutput`），
  不會插在畫面的 escape 序列中間；不呼叫系統的剪貼簿程式，隔著 ssh 也複製得到本機。
- **viewer 的搜尋由 app 做**：terminu 給的兩條路（lib 自己做 `/`，或 app 標範圍、lib 照 S25 畫）選了後者。規範沒寫 viewer 搜尋的
  打字列放哪，lib 自己做就得替規範決定；app 把命中交給 `Viewer.Hits`，打字用一般的 text input（`Search`），kbu、webu 原本的 `/`
  與 `n|N` 照留。
- **最低尺寸預設 80×40**（tdp L1）：用 lib 的 app 在比較小的終端機裡會只看到名稱與提示；版面在更小的畫面也畫得下的 app 自己調低
  （`App.MinWidth`、`MinHeight`）。測試的 app 都調到 1×1，各自的測試照樣在小畫面上跑。

- **tdptest 走 lib 內部的 hook，不開公開 API**（2026-10-20，harness 的請求）：driver 要知道 popup 還在不在播、有幾層、有沒有 toast、
  剪貼簿裡等著什麼，也要把 App 放上測試的時鐘。這些放在 `internal/testhook`（只有這個 module 裡的套件 import 得到），terminu 在
  `init` 填進去（`clock.go`）；app 看得到的只有 `App.Tick`。
- **測試的時鐘，不是真的時間**：真的跑 32ms 一格的動畫，掃一次展示場要幾分鐘，toast 何時消失也跟機器快慢有關；harness 也要能停在
  開場動畫中間（F2）、自己推時間。時鐘從真的時間起算、對齊 loading icon 的一整圈（720ms），app 用真的時間算出來的期限（PIN 的暫停）
  差不多對得上，loading icon 每次都從同一格開始。
- **lib 的計時器與 `tea.Batch`／`tea.Sequence` 做的命令在原地跑**：每個命令都開 goroutine 時，動畫的每一格都要喚醒一次 thread，掃展示場
  的時間大多花在這上面（`-race` 下一個尺寸 15 秒）。這幾種命令一定馬上回來，用 reflect 取函式的 code pointer 認出來（同一個 closure
  的 code pointer 都一樣），其他命令照舊開 goroutine。認不出來也只是變慢，不會錯。
- **`Check` 的沙盒在 `/tmp`，不在 `$TMPDIR`**：macOS 的 `$TMPDIR` 很長（`/var/folders/…`），畫面上印路徑的 confirm、toast 會因此換行或
  截斷，每台機器不一樣（harness 在 hostu 踩過）。app 自己的變數（`<APP>__…`）、家族的（`TERMINU__…`、`LC_TERMINU_…`）清掉，`HOME`、
  `XDG_*`、`USERPROFILE`、`LOCALAPPDATA`、`APPDATA` 指進沙盒，app 就用 E3 的預設位置、落在沙盒裡。

- **app 上色的字另外給 span，不從字串裡分辨**（tdp v0.9.0 #2、#5）：app 自己用 SGR 上色的字與外面帶進來的顏色碼，在字串裡
  長得一樣，lib 分不出來；名字與短字照規範一律不解讀，所以 app 要上色就另外給（`Row.Styled`、`Field.Styled`、
  `FormField.Styled`）。`App.Status` 改成回 `[]Span`、`ColorText` 改成 `ColorSpans`，是故意讓成員編譯不過：留著舊的型別，
  畫面上會默默多出紅色的 `\x1b`。`Row.Cells` 型別沒變（它本來就是字），成員要自己找出塞了顏色碼的格子。
- **lib 自己的名字標記用一個特定的 SGR**：要讓色塊上的名字變成 Red 底，畫到一半的字串裡得認得哪些是名字。用私有區的字元或
  另外的結構都要改動所有量寬度、截斷的地方；一個只有 lib 會寫出來的 SGR（多一個 `24`）走得過既有的每一條路。
- **App Log 的 `!` 只在 panel 上**：K1 的表寫「在 panel 上打開」；popup 裡不作用，跟 `Space` 一樣。
- **app 自己的 popup 有要送出的值時就是一張表單**（tdp popup.8：照表單）：`Popup.Actions` 開的是 `formView`，欄位區換成 app 的
  body（`itemContent` 一個停點）。動作區、`Tab`、按鈕上的鍵、放棄的 confirm、`q`、跑得久的按鈕、錯誤列走表單同一份程式，所以兩邊
  不會各長各的；body 只回答 `Changed`、`Check`，`Focus` 讓它知道要不要畫自己的 cursor。內容上的鍵照舊先給 body，`Tab`、
  `Shift-Tab`、`Esc`、`q` 不給：它們在表單上是固定的。

## 已否決，不要重提

- **bubbletea v2 與 lipgloss v2**：見「設計決定」。
- **用 lipgloss 的 `Style` 排版、`Join*`、`Place`，或 bubbletea-overlay 疊 popup**：icon 佔兩格時框歪，而且換不了寬度函式（C1）。

## 已知的牆與未做

- **畫面自己管的 app 用 `App.Canvas`**：例：locku 的鎖定畫面。整個畫面由 app 畫，lib 不畫畫面、chip 列與 footer，popup 與 toast
  照樣疊在上面，按鍵先給 `CanvasKey`；它一樣走 `App.Run`，renderer 補的「清到行尾」照樣由 `screenOutput` 拿掉。自己開 bubbletea
  program 的 app 沒有那一層輸出，`screenOutput` 沒有開放出去。
- **splash**（Rules S）：機制是 `App.Splash`（`V` 打開、任意鍵關掉、不揭露、只在 panel 上、10ms 的 tick）；`NewSplash` 照 S5 畫：
  讀 app 的 `docs/icon.svg`，一格一個 `nf-fa-square`，下面六列字，揭露動畫全家族一樣。金色的字怎麼分成一個一個字：svg 裡每一個
  XML 註解開始下一個字（`<!-- K -->`），所以 svg 要照 app 名字的順序列字。`Run` 拒絕沒有 splash、或每個 panel 都把 `V` 讓出去的 app。
  展示場的 `examples/showcase/icon.svg` 是 tdp v0.5.4 `principle/icon.svg` 的複本（E8），`examples/minimal/icon.svg` 再複製一份。
- **檔案預覽**（`preview.go`）：照 input/file-picker 的「預覽」表。語法上色用 `alecthomas/chroma/v2`（MIT，catppuccin-mocha，
  只上前景色）；PDF 用 `dslipak/pdf`（BSD-3，rsc.io/pdf 的分支，能取純文字）；壓縮檔用標準庫（zip、tar、gzip、bzip2）。圖片的 data URI
  只讀畫得到的那一段，不把整個檔案（最多 1 MB）都編碼。預覽框不能捲，PDF 讀到預覽區滿了就停，不一定讀滿 20 頁。
- **家族共用的設定 key**（Rules E3）：`hide_secrets_in_preview`、`input_tab_width` 是使用者的設定，lib 不讀設定檔；app 從自己的設定檔讀，
  設到 `App.HideSecretsInPreview`、`App.InputTabWidth`。
- **PTY 只在 unix**：`terminal_other.go` 開不了 PTY，只在畫面上寫出原因。
- **tdptest 還沒做的**（跟 harness 談定的第二版）：檔案檢查（`CheckRepo`：project 範本、README 的 12 段（第 10 段只要連到 CHANGELOG）、`tdp K4`
  的引用、splash 對 `docs/icon.svg`；tdp v0.10.0 起 dev-remarks 與 `docs/<app>-terminu-fix.md` 不再是規定，不查）、`coverage.md` 標上 checklist 的條目編號、自動打開每個動作的 popup、把設定檔改壞再開、檢查 app 沒寫到
  沙盒以外。

## 做不到的規定

（沒有。lib 不自己另做（tdp E8）：做不到或不好用的回報成 tdp 的缺漏，記成盲點，規範改了 lib 才跟著改。tdp v0.10.0 起開發者
文件不在 tdp 的範圍裡（tdp P5），這份備忘照舊留著，給接手的人與 AI 協作用。）

## 設計文件導讀

- [`coverage.md`](coverage.md)：tdp 逐條的清單 —— 規範出處 → 展示場的哪裡、怎麼按到 → 測試名稱。也是展示場「所有的組合」的定義。
- [`walkthrough.md`](walkthrough.md)：手動走一遍展示場的起點 —— 怎麼開、每個畫面看什麼、紀錄的 C2–C4 怎麼驗。
- 規範盲點紀錄：`~/Documents/sideproj/terminu/.local/family-fix/terminu-go/README.md`（C 待驗證、G 盲點）。程式裡的 `TODO(tdp G<n>)`
  對到那裡的編號。
- [`../CLAUDE.md`](../CLAUDE.md)：這個 repo 的任務、規範讀哪些、盲點怎麼記。

## 建置與開發

```sh
go test -race ./...
go vet ./...
gofmt -l .
make showcase    # 用目前的程式碼建展示場（/tmp/showcase）並打開
```

Go 原始碼裡的 Nerd Font glyph 一律寫成 `\U0000E0B0` 這種 8 位數的 code point（tdp E4）。用工具寫檔時，`\uE0B0` 這種 4 位數的寫法
會被當成 JSON 的跳脫字元、變成字面的 PUA 字元。

C1 原型：

```sh
go run ./proto/c1 dump 80 24 menu fits           # 印一個畫面
SHOWCASE__ICON_WIDTH=2 go run ./proto/c1 tui     # bubbletea v1 的 renderer
proto/c1/tmux-check.sh 2 2 tui -- j j p          # 在 icon 佔兩格的 tmux 裡跑、按鍵、擷取畫面
proto/c1/tmux-check.sh 2 2 replay 80 24 j j p    # 同一串按鍵之後的畫面，整張直接印（跟上一行比對）
```

`tmux-check.sh` 要有 `codepoint-widths` 選項的 tmux（在 3.7c 上跑過）。

## 發布

公開在 [github.com/vulcanshen/terminu-go](https://github.com/vulcanshen/terminu-go)，版本是 v0.x 的 tag（v0.1.0 起，對應
tdp v0.6.0）。v0.x 之間 API 還可能變。每一版對應的 tdp 版本寫在 [CHANGELOG](../CHANGELOG.md)。

**push 與打 tag 都先問 user**，每一次都問。

發一版的步驟：

1. CHANGELOG 最上面的 `## Unreleased — tdp vX.Y.Z` 改成 `## vA.B.C — tdp vX.Y.Z`，單獨一個 commit：`vA.B.C: implements tdp vX.Y.Z`。
2. 跑 `go test -race ./...`、`go vet ./...`、`gofmt -l .`。
3. 打 annotated tag，先推 main、再推 tag：

   ```sh
   git tag -a vA.B.C -m "terminu-go vA.B.C — tdp vX.Y.Z"
   git push origin main && git push origin vA.B.C
   ```

4. 在 repo 外面建一個乾淨的 module，確認 proxy 抓得到、build 得過：
   `GOWORK=off go get github.com/vulcanshen/terminu-go@vA.B.C && GOWORK=off go build ./...`。
5. 通知用 lib 的 app（kamonu、ccdrawu）：這版改了什麼、要不要改呼叫的程式、怎麼升級。

踩過的坑與定下來的做法：

- **tag 一推出去就被 proxy.golang.org 永久快取**：不搬、不刪、不重打。發錯了就發下一版。
- **tdp 發版、lib 卻不用改時不單獨發**（只動了文件裡寫的 tdp 版本）：commit 留在本機，跟下一個有改 lib 的版本一起發。
  CHANGELOG 那一節維持 `Unreleased`，發版時把版本拉到最新的 tdp。例：tdp v0.6.4 的跟進是跟著 v0.1.4 一起出去的。
- **README 與 CHANGELOG 寫的 tdp 版本，要先在 terminu 的 GitHub 上有那個 tag**，別人才查得到。tdp 還只在本機時先別發。
- **用 lib 的 app 不用 `replace`**：go.mod require 發布的版本；本機用 gitignore 掉的 `go.work`（`go work init . ../terminu-go`）
  接上這個 repo 的工作目錄。app 發版前用 `GOWORK=off go build ./... && GOWORK=off go test ./...`，確認只靠發布的版本也過。
