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
24-bit 色（truecolor）。終端機元件（在 PTY 裡跑程式）能在 Unix 與 Windows 10 1809 以後（ConPTY）上用。

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
	"fmt"
	"os"
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
		fmt.Fprintln(os.Stderr, "todo:", err) // errors on stderr after "<app>: " (tdp E1)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "todo:", err)
		os.Exit(1)
	}
}
```

`x` 把 cursor 所在的待辦標成完成。`Space` 列出這裡能做什麼，`?` 列出按鍵，`V` 打開 splash，`q` 離開。每個 terminu app 都有
一個畫著自己 mark 的 splash（tdp S1、S5）；`icon.svg` 就是那個 mark，這個範例用的是 terminu 的副本。

直接關掉終端機視窗時，`Run` 跟離開一樣回來、回 nil，`main` 在它之後做的事（例如存檔）照樣會跑。app 的終端機裡在跑的程式
會先被掛斷，跟關掉終端機視窗一樣，所以 `Run` 之後不會一直等它們結束。那時終端機已經不在了，寫什麼上去都看不到。

想看它做得到的全部，跑展示場（用假資料展示每一個元件的範例 app）：

```sh
go run github.com/vulcanshen/terminu-go/examples/showcase@latest
```

## 文字與顏色

交給函式庫的字串都是文字。從外面來的東西（檔名、網頁標題、程式的輸出）一律不原樣送給終端機：

- **名字與短字**（清單的格子、表單與 panel 上的值、menu 的 label、標題、toast、confirm、選項、statusbar）：每個控制字元都畫成
  紅色的名字，Tab 是 `\t`，顏色碼是 `\x1b` 接著 `[31m`。要自己上色時另外給 span：`Row.Styled`、`Field.Styled`、
  `FormField.Styled`，`App.Status` 也回 span。內容有自己顏色的列（例：排行的名次），可以用 `Row.CursorColor` 讓停在上面的
  cursor 改用那個顏色當底（tdp color.2）：只能是那一列自己的顏色；模式裡 cursor 照舊是 Subtext1。在 `Canvas` 或自己的
  popup 裡畫外面來的字時，用 `Escape` 照同一套畫。
  清單裡放一行一行文字的欄（原始碼的一行、log 的一行）標成 `Column.Line`：裡面的 Tab 畫成空白。
  自己寫的字不放換行字元：好幾列的訊息一列一列給，`app.Error(title, row1, row2)`、`app.Toast(kind, row1, row2)`，
  畫面上紅色的 `\n` 就一定是外面來的。
- **拿來讀的內容**（viewer、panel 上的 log、預覽）：照終端機會畫的樣子，顏色碼照著上色、`\n` 換行、進度列的 `\r` 只留最後的
  狀態，其他畫成紅色的名字。自己畫的 body 要顯示 log 時，用 `Reading` 畫它的每一行。
- **從標記語言畫出來的文件**（網頁、畫成文件的 Markdown 檔）：照瀏覽器的樣子畫。函式庫沒有畫這種文件的元件，viewer 與預覽
  顯示的是 Markdown 檔的原文。自己排版時，一般文字裡的 Tab 與 `\n` 換成空白；程式碼區塊裡的 Tab 換成 `App.InputTabWidth`
  格空白（沒設是 4）、照 `\n` 分列；再用 `Escape` 畫，其他控制字元（連顏色碼的 ESC）都是紅色的名字，顏色不從文件來。
- **語法顏色**：`Lang` 寫了語言的 viewer、預覽裡的文字檔，照 tdp 上色（檔案的格式照檔名判斷，同名的再看資料夾：`.ssh` 裡的 `config`
  是 SSH 設定，`hosts` 只有在 `/etc` 裡才是 hosts）：chroma 的 catppuccin-mocha，照它的顏色、粗體、斜體與底線，不上背景色，chroma 漏掉的照
  家族的一種補法補上（TOML 的表名與 key、YAML 沒引號的值……）。自己畫的 body 要顯示檔案時，用 `Syntax(lang, lines)` 上色，
  長得就跟預覽一樣。import 函式庫時，它會把自己的 lexer 放進 chroma 的全域 registry，你自己呼叫 chroma 也拿到補過的。

## App Log

在任何 panel 上按 `!` 打開 App Log：app 做過的事與出問題的事，最新的在最下面，只放在記憶體裡。每一個 toast
與 error popup 都會自己記進去；其他的用 `app.Log(kind, text)` 記（動作的結果、app 在背景做的事），
任何 goroutine 都能叫；寫法跟 toast 一樣，大寫開頭。有沒看過的錯誤或警告時，footer 的 `!:log` 會寫出有幾個。

## 讀不了自己的檔時

app 讀不了自己的設定、狀態或資料（格式錯、檔案壞掉、沒有權限）時，在啟動時叫 `app.MoveAside(path, err, startFresh)`。
使用者會被問要不要把檔移開：`Enter` 改名成 `.bak`（一律不蓋掉別的）再跑 `startFresh`，app 在裡面像第一次打開一樣開始；
`Esc` 離開，`Run` 回傳路徑與原因，交給 `main` 印出來。資料是一筆一個檔時，讀不了的那一筆改叫
`app.MoveAsideRecord(path, err, reread)`：`Enter` 只把那一筆移開，再跑 `reread` 讀其他的。改版後不再用的檔交給使用者處理：
每次啟動叫 `app.NoLongerUsed(paths...)`，還在的那幾個會在 App Log 各記一筆。

沒有畫面時（例：印到 pipe 的指令），這一節說的每一件都寫到 stderr，一件一行，開頭 `<app>: `，跟 TUI 同一句話；TUI 會繼續的
地方，指令也繼續，exit status 看指令自己的事有沒有做完（tdp E3、E1）。讀不了的檔照 `Esc` 做：不動任何檔案，寫出
`terminu.CannotReadText(path, err)`（跟 `Run` 回傳的同一句：`~/.config/catu/settings.yaml can't be read: …`），想說怎麼辦可以
多一行（例：`Open catu to move it aside and start fresh`），以 exit status 1 結束。`terminu.NoLongerUsedLines(paths...)` 是
`NoLongerUsed` 記的那幾行。

讀得進來、只是有 app 用不了的值（不是 `#rrggbb` 的顏色、負的秒數），或有不合規則的一筆（名字空白或重複）的檔，不算讀不了：
app 繼續。啟動時先用 `copied, err := terminu.KeepAside(path)` 把檔複製一份（`.bak`，一律不蓋掉別的），再用預設值重寫
那些值、拿掉那些筆，然後叫 `app.DidNotFit(path, copied, misfits...)`：一個 Warning toast 寫出檔案與備份，每個
`terminu.Misfit{Key, Was, Now}` 在 App Log 記一筆 WARN。沒有畫面時，把
`terminu.MisfitNotice(path, copied, misfits...).Lines()` 寫到 stderr：toast 那一行，再接每一筆 WARN。`KeepAside` 失敗時不重寫檔案：預設值只用在這一次，把它的 error 用 Error toast（或 stderr）說出來，再叫
`app.DidNotFit(path, "", misfits...)` 記 WARN。

值指到使用者自己資料裡不在的一筆（例：某個設定指向手動刪掉的 profile），不算值不合規則：不叫 `KeepAside`，檔案不動、值留著。
這一次用 app 內建的預設值，啟動時用 `app.Toast(terminu.Warning, …)` 說出來；每個 toast 都照同樣的字記進 App Log，那就是
那筆 WARN，不用另外叫 `app.Log`（沒有畫面時，把那句寫到 stderr）。畫面上照原本寫的顯示，讓那一列的 `Err` 回傳 true，label
就是 Red；那一筆不在時，它的 select 用 `Selected: -1` 打開。

同一次啟動要說的不只一件時（位置搬了、一個或好幾個檔有值不合規則、備份做不出來、值指到不在的一筆），每一件各是一個 toast，
跟所有 toast 一樣往上疊：照發生的順序，每一件收成一個 `terminu.Notice{Kind, Text}`，檔案裡不合規則的值改用
`terminu.MisfitNotice(path, copied, misfits...)`（不叫 `DidNotFit`），最後叫 `app.Notify(notices...)`。一件一個 toast；
App Log 一件一筆、用它自己的種類，那個檔的 WARN 緊接在它後面。沒有畫面時，改把每個 notice 的 `Lines()` 寫到 stderr，一行一行。

toast 會往上疊：新的出現在最下面，把正在顯示的往上推；每個時間到就走；最多顯示三個，第四個來時最舊的先走（它的話在 App Log
裡）。跟正在顯示的某個 toast 一模一樣的話（同一種）不會多一個：那個重新計時，標題寫出次數（`INFO ×3`）。`Esc` 一次收一個，
從最舊的開始。

啟動之後才來的（例：畫面開了之後才問得到叢集的 namespace、種類）也交給 `app.Notify`，一樣各是一個 toast、跟其他的疊在一起，
所以不會在被讀到之前被另一個蓋掉。

設定檔裡 app 不讀的 key 一律留著，不管是改版後不再讀的，還是從來沒讀過的（例：打錯字的 `refesh`），app 重寫那個檔時也留著：
不跳 toast、不備份、不收成 `Notice`。每次啟動用 `app.Log(terminu.Info, …)` 各記一筆 INFO：app 知道自己以前讀過的寫
`No longer used: measure in ~/.config/app/config.yaml; delete it if you don't need it`，其他的寫
`Not read: refesh in ~/.config/app/config.yaml; delete it if you don't need it`。沒有畫面時，這幾行寫到 stderr。

TUI 以外（app 自己畫、不在 TUI 裡的畫面，或印給別的程式顯示的輸出，例：狀態列），上面這些都不做：不叫 `MoveAside`、
`KeepAside`、`DidNotFit`、`Notify`，也不動任何檔案（不移到 `.bak`、不重寫）。這一次用預設值（讀不了的檔要怎麼繼續由 app
決定），並在那裡讓使用者看到哪個檔有什麼問題，怎麼呈現由 app 決定（例：狀態列最後加一個標記、側邊欄多一行）：那裡沒有人看
stderr。下次打開 TUI 時，照上面做。

## 成功的報告

使用者需要讀完、toast 又放不下的成功報告（例：匯出寫了哪些檔、寫在哪裡），放進唯讀的 viewer：
`app.Report(&terminu.Viewer{Title: "Exported", Lines: lines})`。它是自己跳出來的，所以 `Enter` 跟 `Esc` 都能關；也會以
Success 記進 App Log。

## 要提權時

要以管理員身分做事（例：把暫存檔複製到 `/etc/hosts`）時，叫 `app.Elevate(terminu.Elevate{Command: ..., Done: ...})`，只給要跑的
指令。不用密碼時直接跑；要密碼時 lib 開 `PASSWORD` 的 password popup，把密碼交給 sudo，打錯了留在 popup 裡重打。app 拿不到密碼：
它不會被存下來、記進 App Log、出現在畫面上或送出去。`Done` 拿到指令怎麼結束的：失敗時是 `*terminu.ElevateError`
（指令印出來的每一列），交給 error popup；使用者取消時跑 `OnCancel`。沒有 sudo 只有 doas 時設 `Tool: terminu.Doas`，doas
在 terminal popup 裡自己問密碼。Windows 不用這個，照舊用 gsudo 或 Windows 的 sudo（系統跳 UAC）。

## Diff

`app.DiffModeAction(panel, list, terminu.DiffMode{Key: "D", Text: ...})` 讓清單有 Diffing 模式：使用者挑兩列，在 Diff 裡看哪裡不一樣，
UNIFIED 或 SPLIT。兩列都裝著許多項目時（兩個資料夾、兩個 namespace、兩個 commit），改成進到一層「有差異的項目」清單：
`Many` 說哪些列裝著許多項目，`Items` 給出兩列之間有差異的項目，一項一個 `terminu.DiffItem`。那層清單由 lib 畫在 panel 裡，
每一項前面的標記（`DiffAdded` +、`DiffRemoved` -、`DiffChanged` ~）、`Enter`、`Esc` 與位置列也都是 lib 的。
舊的名字 `CompareAction` 與 `Compare` 照樣編得過，之後的 minor 版拿掉。

`DiffItem` 用 `Anchor` 與 `Other` 寫出兩邊的名字，舊的那邊當 anchor，名字要看得出在哪裡（`ui.go @ 1a2b3c4`）：不管從哪裡
打開，它的 Diff（標題是 `DIFF`）第一列都寫這兩個名字，跟位置列一樣緊貼上框，太長時折行、不截，它進去的那一層清單也用它們當位置列。它有 `Text`（兩邊的文字，沒有的那邊給 ""），或是它
自己也裝著許多項目時的 `Items`；兩個都沒有，就是不是文字、沒有 Diff 的項目。app 知道是改名的，用 `DiffChanged`，名字寫
`舊 → 新`。兩邊一樣的不要列，文字等到要比時才讀。

app 自己的動作要顯示同一項的兩個版本時（commit 裡的一個檔對它前一個 commit、正在改的檔對它最後一次 commit），用
`app.OpenDiff(panel, item)` 打開同一個 Diff：有 `Text` 就是那個 Diff；有 `Items` 就是它裡面有差異的項目清單，畫在 panel 裡
取代原本的列，按 `Esc` 回到原本的列。app 自己的動作要在那個 panel 顯示別的東西時（例：在分支上按 `Enter`，把它的 commit
放到 cursor 下），先叫 `panel.LeaveDiff()`：離開每一層這種清單，像一直按 `Esc` 到原本的列回來為止；沒有清單時什麼都不做。

進到這種清單會換掉 panel 列的東西，所以 `OpenDiff` 與 Diffing 模式進去之前會先清掉 panel 的篩選（tdp panel.14）。app 自己的動作要換掉
panel 列的東西，或把它的 cursor 放到某一項上時（例：在別的 panel 移動 cursor，把這個 panel 帶到那一項；進到一項裡面），也一樣
先叫 `list.ClearFilter()`，不管篩選有沒有藏住那一項。只是 focus 離開 panel，篩選留著。

顯示差異的 panel（例：跟著 cursor 的明細，顯示 cursor 上那個檔改了什麼）畫法跟 UNIFIED 一樣，不照工具印的樣子（`git diff`
的標頭與 `@@` 那幾行）：app 自己讀兩邊的文字，在它的 `Pager` 上叫 `pager.SetDiff(anchor, other)`，再給 panel 一個 `Enter`
動作，用 `app.OpenDiff(panel, item)` 打開 Diff。兩邊一樣時，顯示 Pager 的 `EmptyFact`。

顯示的東西裝著許多項目時（例：新的資料夾、一個 commit），要列出有差異的項目，不能只寫名字：`pager.SetDiffItems(items)` 每一項
先畫它的標記與顏色、再畫名字，跟 `OpenDiff` 進去的清單一樣（新資料夾的項目全部是 `DiffAdded`）。它的 `Enter` 在這個 panel 裡
進到那份清單：`app.OpenDiff(panel, terminu.DiffItem{Anchor: ..., Other: ..., Items: ...})`，給同樣的項目。一項都沒有時，顯示
Pager 的 `EmptyFact`。

有明細 panel（`panel.Details`）的清單，可以改把 Diffing 模式放在明細裡，不打開視窗、也不進到清單裡：設 `InDetails`（用哪一種由 app
決定）。這時模式同時涵蓋清單與明細，兩個框都是模式顏色、都寫 `Diff`。明細跟著 cursor，不用按鍵：lib 用 app 的 `Text` 與
`Items`，在明細裡畫出 anchor 跟 cursor 那一列的差異，畫法跟 `SetDiff`、`SetDiffItems` 一樣；cursor 在 anchor 上時是空狀態
（`Move to another <Noun> to diff with <anchor>`：設 `Noun`，沒設時是 `item`），跟 anchor 一樣的列也是。在清單上按 Diff 鍵
或 `Enter`，focus 移到明細，在那裡打開一項的差異或再進一層；`Esc` 照使用者來的路退回，先回到清單、anchor 留著，再離開模式，
明細換回 app 自己的內容。模式裡明細原本的內容、位置列、`Fail` 與 loading 那句都不顯示，照常更新就好。清單的 `Details` 一定要是
同一個畫面的另一個 panel。

## 占好幾列的項目

清單的項目可以在第一列下面再多幾列，只給需要的項目（例：commit 的分支與 tag 放在它下面）：`Row.Below` 是
`[]terminu.BelowRow`，每一列是一排放在清單欄位底下的 `Cells`，要上色用 `Styled`，跟 `Row.Styled` 一樣。格子給 `""` 那一欄就留白，
所以一列的字從哪裡開始由你放；有字的最後一格一直延伸到列尾。欄位、表頭與勾都在第一列。cursor 畫在第一列，下面幾列跟著亮。
移動、`N of M` 與捲動都算項目，panel filter 比對項目時每一列的字都算（tdp panel.5）。

## 寬度有剩時

終端機很寬、每一欄都已經是最寬的值、也沒有收掉的欄時，`List.Spread` 決定剩下的寬度放哪裡，每個清單由你挑（tdp panel.5）：
`SpreadEnd`（預設）放在列尾；`SpreadFill` 全部給標了 `Column.Fill` 的那一欄（長的那一欄，例：說明），後面的欄靠到最右邊；
`SpreadEven` 先撐寬最窄的幾欄，直到盡量一樣寬；`SpreadGaps` 每個欄距一起從 2 格加到最多 6 格，還有剩給 `Fill` 那一欄，沒有就放列尾。
空間不夠時四種都一樣，照原本的先縮、再收掉欄。

## app 自己的模式

`panel.EnterMode(&terminu.Mode{...})` 讓 panel 進入 app 自己的模式（tdp K11）：只有模式的 `Keys` 與 core key 作用，`Esc` 離開。
名字照 tdp 的模式取：在模式裡正在做的事的 -ing 字，一個詞（`Playing`，跟 `Dragging`、`Viewing` 一樣）；`panel.EnterLayer`
再進一層，框上用 `:` 把名字接起來（`Viewing:selecting`，tdp panel.3）。
`ModeKey` 只收 `Keys` 列出來的鍵，lib 不會替你補：方向鍵與 `PgUp|PgDn` 照 tdp K12 自己列進去；key reference 與 footer
寫的是 `Key`，只寫字母：

```go
{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
```

模式的 Space menu 照 tdp menu.4 列出模式的鍵。app 自己的模式在那裡列什麼，由 app 決定（tdp K11）：`NoMenu` 讓一個鍵不列進
menu，`Mode.NoLeaveRow` 不列離開模式那一列。每個鍵照樣列在 `?`，footer 不變，`Esc` 照樣離開。什麼都沒列的 menu 照樣列出
離開那一列，因為模式的 Space menu 至少要有一列。

那個 menu 的一列先關掉 menu，再做那個鍵做的事，所以鍵開出來的 popup（例：重新開始前的 confirm）取消或關掉後回到模式，
不回到 menu（tdp K11）。popup 本身照它的元件畫、照它的元件答：confirm 照樣用 `Enter` 與 `Esc` 回答。

離開 app 自己的模式時，`OnExit` 可以跳出一個 popup，說離開後是什麼狀態（例：暫停後用 confirm 問要不要繼續，tdp K11）。
popup 出現時模式已經離開：在它上面按 `Esc` 是關掉它、留在模式外面；要回到模式，是 popup 自己的動作（`EnterMode`）。
用模式的鍵、或模式自己結束（例：game over）時，用 `Panel.LeaveMode` 離開：模式的 Space menu 與 `?` 一起關掉，接著 app 開的
popup 就是唯一的一個（tdp K11）。

讓使用者改 app 自己的模式的鍵時（tdp K11），`Enter` 與移動鍵也可以改。使用者選的鍵放進 `Key` 與 `Keys`，方向鍵與
`PgUp|PgDn` 留在模式預設給它們的那個 `ModeKey` 的 `Keys` 裡。左移改成 `a` 就是 `{Key: "a", Keys: []string{"a", "←"}}`，
`?` 寫 `a|←`，footer 與 Space menu 寫 `a`。一對（一個方向一半）的一半被使用者改掉時，`?` 要拆成兩列（tdp M5）：寫在
`Ref` 裡，footer 照樣把那一對寫成一項。

```go
{Key: "Enter|k", Keys: []string{"Enter", "k", "PgDn", "↑"}, Desc: "rotate", Do: rotate,
	Ref: []terminu.KeyRow{{Key: "Enter|PgDn", Desc: "rotate left"}, {Key: "k", Desc: "rotate right"}}},
```

使用者也可以把模式的一個動作放到 `Space`，但 app 不能預設放上去（tdp K11）。寫法是一個有 `Do`、收 `Space` 的
`ModeKey`，例：`{Key: "Space", Desc: "drop", Do: drop}`：`Space` 就做那個動作，這個模式沒有 Space menu。footer 把它寫在
最前面、原本 `Space:menu` 的位置（`Space:drop ?:help …`），`?` 把它列在模式的鍵裡。原本只列在那個 menu 裡的鍵，這時要
放進 footer：拿掉它們的 `NoFooter`。`Space` 已經有動作時擋下、跳 Warning 說 `Space` 改做什麼與要 menu 先按 `Esc`、
tdp 的模式（Dragging、Diffing）不讓放，都是 app 的事。收 `Space` 但沒有 `Do` 的 `ModeKey` 照樣是 core key，`Space` 開 menu。

app 自己的模式的 footer，在 `Space:menu ?:help` 與 `Esc:<動詞>` 中間寫哪些鍵由 app 決定，core key 也可以（tdp K11、
screen.2）。要寫 `q:quit`，給模式一個沒有 `Do` 的 `ModeKey`：`{Key: "q", Desc: "quit"}`。`q` 照樣進入離開流程，跟在任何
模式裡一樣，Space menu 裡它那一列也一樣；`?` 只在 core keys 列它一次。

## 從 app 自己的 popup 執行動作

app 自己的 popup（`App.Popup`）上的鍵可以執行 app 的動作，跟從 menu 執行一樣（tdp popup.8）：`App.RunAction(key)` 執行
在 focus 的 panel 上（模式外）按 `key` 會跑的那個動作，可以是項目或 panel 的操作、panel 的 `Enter` 動作或 global 操作；
`q` 是離開流程。要先問的動作，confirm 開在 popup 上面；停用或隱藏的動作什麼都不做，跟按了沒有動作的鍵一樣。動作跑完、
沒有再開 popup 時，所有 popup 一起關掉。

```go
func (b *settings) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	if key == "n" { // the panel's [n]ew game
		return true, b.app.RunAction("n")
	}
	return false, nil
}
```

popup 開在 app 自己的模式裡的 panel 上時，動作真的執行時才離開模式：有 confirm 的話是按下 `Enter` 之後，所以在 confirm
上按 `Esc` 什麼都沒變。離開的方式跟 `Panel.LeaveMode` 一樣，不跑 `OnExit`：`OnExit` 原本會做的事（例：暫停）要在呼叫前
自己先做；離開模式會丟掉使用者做出來、要留下的東西時，在呼叫前自己先問（tdp K11）。`q` 不離開模式，離開流程會先問。

## 項目的附註

項目可以有一則附註：連結會去哪、備註（tdp panel.18）。`panel.Note` 設成一個 func，回傳 cursor 那一項的附註，沒有就回 nil
（app 自己的模式裡用 `Mode.Note`）。cursor 停在有附註的項目上時，附註的頭出現在那一項正下方：一個小框，裡面是附註的 glyph 與
`i`；按 `i` 打開，一個標題 `NOTE` 的 viewer 貼在那一項下方（那一項在 panel 最底下一列時開在上面），最高到 panel 高度的一半，
有 viewer 的捲動、選取模式與 `/` 搜尋；`Esc` 收回成頭。打開時 lib 自己記一筆 App Log（`Opened the note of <那一項>`），app 不用另外記。lib 會把 `[i] Note` 放進那個 panel 的動作；`i` 是全家族打開附註的鍵
（tdp K13），所以那個 panel 上 app 自己的動作不能用 `i`。`List` 自己說得出 cursor 那一項在哪；app 自己的 body 要實作
`terminu.NoteSpot` 的 `NoteAt() (top, bottom, col int, ok bool)`。

```go
pods.Note = func() []string {
	if r, ok := list.Current(); ok && r.Cells[2] == "Pending" {
		return []string{"0/3 nodes are available: 3 Insufficient memory"}
	}
	return nil
}
```

附註取代的 inspect toast（`Panel.Inspect`、`Mode.Inspect`）暫時照樣能用，之後的 minor 版拿掉。

## 跟著帶頭 panel 換的 panel

畫面上有些位置可以跟著「幾塊帶頭的 panel 裡，最後拿到 focus 的那一塊」換（tdp L5）：例如 git 工具的右邊，帶頭的是分支時顯示
commit，是改動的檔案時顯示那個檔的 diff 與 history。跟著的位置寫在 `Screen.Following`（`Panels` 的索引），每一塊帶頭的 panel
要顯示什麼寫在 `Screen.Leads`：`terminu.Lead{Panel: changes, Shows: []*terminu.Panel{diff, history, nil}}`，`nil` 是收起那個位置。
focus 移到帶頭的 panel 時，lib 把它的那幾塊放進那些位置；focus 移進它們時什麼都不換。號碼照位置：一個位置的 `[N]` 不管顯示什麼
都不變，收起的也是，收著時它的數字鍵不作用。app 的 `Layout` 用 `screen.Hidden(i)` 把收起那塊的空間給旁邊。

## 說明牌

只用來說明的 panel（例：遊戲旁邊的 NEXT、INFO、KEYS，圖例）是說明牌（tdp panel.17）：`Sign` 設成 `terminu.SignLit`
（常亮，跟串流內容一樣不淡化）或 `terminu.SignDim`（常暗，跟沒有 focus 的 panel 一樣淡化），跟其他 panel 一起放進
`Screen.Panels`，一起排版、一起縮放。它永遠拿不到 focus：`Tab` 與數字鍵跳過它，框上只寫 label，其他 panel 自己從 `[1]`
編號。它沒有動作、tab、hint、捲動位置（有 `Actions` 或 `Tabs` 時 `AddScreen` panic）。它不能捲動：內容要在最小畫面
（60 × 32）分到的大小裡放得下；`Pager` 的行還是放不下時，從底部切掉，看得到的最後一列結尾加 `…`。

```go
keys := &terminu.Pager{}
keys.SetLines([]string{"h|l  move", "k    rotate"})
sign := &terminu.Panel{Label: "Keys", Body: keys, Sign: terminu.SignLit}
```

## 移動 panel 之間的線

使用者在 panel 上，從有 focus 的那個移動 panel 之間的線（tdp screen.9）：`>`、`<` 讓它寬或窄 2 欄，`+`、`-` 高或矮 1 列，
`=` 讓這個畫面的線回到你的 `Layout` 的樣子。lib 從 `Layout` 找線：panel 共用的邊，panel 沿著它接多遠，線就多長，接著它的
panel 一起動。panel 最小能多小用 `MinWidth`、`MinHeight` 給（沒給是 10 × 3）；60 × 32 時所有 panel 仍要並排（tdp L1）。這五個鍵
在 panel 上是 lib 的：`Run` 不讓 panel、畫面或 global 的熱鍵用它們（mode 與 app 自己的 popup 照樣可以用）。線停在哪裡由 app
記在自己的 state（tdp E3），依畫面與終端機大小：`Run` 之前讀進 `App.Lines`，`App.OnLines` 拿到的寫回去。使用者移動線之前不會叫它。

```go
app.Lines = state.Lines // a terminu.Lines, as read from your state file
app.OnLines = func(l terminu.Lines) tea.Cmd {
	state.Lines = l
	if err := state.Save(); err != nil {
		return app.Toast(terminu.Error, "state.yaml could not be written: "+err.Error())
	}
	return nil
}
```

## 只有一個 panel 的畫面上的數字鍵

在 panel 上，數字鍵 `1`–`9` 用來跳 panel（tdp K1）。畫面上只有一個 panel 有號碼時（說明牌沒有號碼），`2`–`9` 沒有 panel
可跳，`1` 就是它自己，這時 `Panel.Digits` 可以把數字鍵拿去用，例如翻到第幾頁。lib 把範圍寫進它的 hint，放在 `Enter` 與 `h|l`
後面（`1–3:page`；超過九個寫 `1–9`，只有一個就不寫），`?` 寫在移動那一區；Space menu 沒有這一列。在模式裡，數字鍵跟在一般
panel 上一樣；在 filter 打字時照樣打進去。畫面上還有別的 panel 有號碼時，`Run` 不接受 `Digits`；之後多出一個（執行中加上的，
或被帶頭 panel 藏起來的位置，tdp L5），它在的期間數字鍵回到跳 panel。使用者在哪一頁由 body 的 `Scroll()` 寫，例如 `2 of 3`。

```go
pins := &terminu.Panel{Label: "Pins", Body: grid, Digits: &terminu.Digits{
	Count: grid.Pages, // 3 pages: 1–3:page
	Hint:  "page",
	Desc:  "go to page N",
	Do:    func(n int) tea.Cmd { grid.ShowPage(n); return nil },
}}
```

## panel 框上的狀態

panel 可以在框上放一個 app 自己、一直在的狀態：一個 glyph 加一段短字，例如 panel 監看的檔案最後一次讀進來的時間
（tdp panel.1）。要不要放、寫什麼、放哪個角，由 app 決定：

```go
logs.FrameStatus = terminu.FrameStatus{Glyph: glyphWatch, Text: "17:43:42", Corner: terminu.BottomRight}
```

lib 把它畫在那個角原有的東西靠中間的一側：膠囊串和 loading icon 後面、模式名左邊、hint 後面，或搜尋的狀態和捲動位置
左邊。顏色跟框線一樣；它講的事出了問題時，把 `Kind` 設成 `terminu.StatusWarning`（Peach）或 `terminu.StatusError`
（Red）。框放不下時它最先拿掉、整個拿掉，有位置了再回來。它講的事變了，就在 update loop 裡改，例如在自己的計時器裡
（`App.Tick`，訊息進 `OnMsg`）；設成零值就拿掉。popup 沒有這個狀態。

## 開著時會變的內容

要不要重新讀來源（例：app 監看的檔案）、什麼時候讀，由 app 決定；顯示它的地方怎麼跟著變照 tdp note.9，lib 會做：

```go
cmd := app.ReadAgain(viewer, lines) // 開著的 viewer
pager.ReadAgain(lines)              // panel 的 Pager，要設 Follow
```

`ReadAgain` 等於在新的內容上重新打開 viewer：原本在結尾就跟到新的結尾（選取模式裡不跟），行號欄只變寬，搜尋重新找，
選取模式在位置還在的地方留著；回傳的是說明拿掉了什麼的 toast，選取模式的 Space menu 或 `?` 開著時則是疊在上面、跟它們
一起關的 note。來源讀不到時呼叫 `app.CannotReadAgain(viewer, "Cannot read notes.txt: it is gone")`：app 的字放進 Error
toast，或疊在那個 menu 上的 error popup。app 自己的閱讀 panel 有自己的選取模式時，選取失效用
`app.SelectionGone(panel, terminu.Warning, "Read again", "notes.txt changed: Viewing left")` 說，在離開模式之前叫：是 toast，
或疊在模式的 Space menu、`?` 上的 note。開著時會給新的行的 Pager（`Append` 或 `ReadAgain`）要設 `Follow`。重新讀進來的內容不算串流：
不要設 `Panel.Stream`（tdp T2）。

這種重讀讀到什麼，App Log 只記一次（tdp note.7）。讀到變了，記一筆 `app.Log(terminu.Info, "Changed outside the app: branch feat deleted")`
寫出改了什麼；讀到沒變什麼都不記。讀不到時，第一次跳 Error toast（toast 自己會記）；之後還是同一個失敗（同一句話），不跳也不記，
這時叫 `CannotReadAgain` 不要給字；換了一個失敗，再跳一次。又讀得到時記一筆 `app.Log(terminu.Info, "The repository can be read again")`，
不跳 toast。自己一直進來的更新（串流、連線一直保持最新的清單）不一筆一筆記。

## 命令列

每個 app 的指令印出來的東西都照同一種格式（tdp E1）：結果寫到 stdout；錯誤、警告寫到 stderr，一件一行，開頭 `<app>: `，
跟同一件事的 toast 同一句話；用錯（不認得的指令、少了參數）寫到 stderr：`<app>: ` 接哪裡錯（`Unknown command "x"`）、空一行、
再接用法，exit status 2；做完 0、沒做完 1。有 `<app> iconwidth` 的 app，把 `terminu.Probe(name).Lines()` 印到 stdout：
一個 icon 和一個 powerline 符號各占幾格，後面寫數字從哪來（`icon 2 (probe)`、`powerline 1 (TERMINU__POWERLINE_WIDTH)`）。

每個 app 都有 `<app> update`：照當初安裝的方式，把 app 更新到最新的正式版（不更新到 beta）。每個 app 都一樣，所以由 lib 做：

```go
case "update":
	os.Exit(terminu.SelfUpdate{App: "todo", Version: version.Version, Owner: "vulcanshen", Tap: "vulcanshen/tap"}.Run(os.Stdout, os.Stderr))
```

用 Homebrew 裝的，執行 `brew upgrade <Tap>/<App>`；用 install script 裝的、或從 release 下載的，從 `github.com/<Owner>/<App>`
下載這個系統與架構的新版，對過 release 的 `checksums.txt` 再換掉正在執行的檔；用 Scoop 裝的，印出要執行的 `scoop update <app>`。
已經是最新版時說一聲、exit 0；從原始碼編的（`Version` 是 "dev" 或空的），或沒有權限寫入檔案所在的地方時，說明原因與怎麼做、
exit 1，不會要更多權限。release 的檔名要照家族 goreleaser 範本的命名：`<app>_<version>_<os>_<arch>.tar.gz`（Windows 是
`.zip`），裡面有執行檔，另有 `checksums.txt`。`Source` 寫從原始碼編的怎麼更新（`git pull && make install`）；`Then` 是使用者
自己更新完之後要跑的指令；`After(exe, updated)` 是 app 到了最新版之後自己要做的事，更新過時 `exe` 是新的執行檔。

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
- **畫面上有什麼**：`Screen` 是每一列的文字，`Styled` 帶顏色，`Has` 找字，`Quit` 回報 app 要離開了沒（之後跟真的程式一樣，什麼都不再送給 app：離開時正在收起的 popup 就停在那裡），`Clipboard` 是它複製了什麼。
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
  它不讀 app 的 repo：家族的範本與 repo 的檢查在 [terminu-harness](https://github.com/vulcanshen/terminu-harness)。
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

每次輸入播動畫時，測試的時間也跟著往前走那麼久，就像使用者等動畫播完；`Advance` 途中到時的計時器在它自己的時間觸發，
之後時間才走到 `Advance` 的終點。所以 app 在計時器到時記下的時間（例：最後一次讀檔的時間）會比之後的 `d.Now()` 早，
最多早那段 `Advance` 加上之後播的動畫：測試時檢查它落在前後兩次 `d.Now()` 之間，不要求等於後面那一次。

畫面會自己動的（例如遊戲的時鐘），在使用者看不到畫面時要停住：每次計時器到時問 `app.Covered()`。有 popup 蓋著畫面（正在收起的
算到它完全不見）、或開著 splash 時，它回 true；toast 不算蓋著。

## lib 與 app 的分工

tdp E8 對任何共用的 lib 只留三點：用 lib 的 app 不另外做 lib 做到的、畫面測試照誰畫的分、lib 不自己另做。這個 lib 跟用它的 app
怎麼分工，照這張表：

| 規定 | lib | 範例 app | 用 lib 的 app |
|---|---|---|---|
| 畫面的規定（K、M、L、F、X、T、S1–S4、components） | 做到 | 全部展示出來 | lib 給的，自己不另外做 |
| E1 `version`、`help` | — | 要 | 自己做 |
| E2 環境變數的命名 | lib 自己讀的變數掛在 app 的名字底下（例：`<APP>__ICON_WIDTH`） | 照 E2（例：`SHOWCASE__…`） | 自己的變數照 E2 |
| E3 設定與資料的位置 | —（使用者移動過的 panel 之間的線，從 `App.OnLines` 交給 app） | 不存東西，不適用 | 自己做，`App.Lines` 也在裡面 |
| E4 需求 | README 寫明 | — | README 寫明 |
| E5 icon 的實際寬度 | 做到：探測（icon 與 powerline 符號）、顯示寬度函式、替子程序設 `TERMINU__ICON_WIDTH` 與 `TERMINU__POWERLINE_WIDTH`、啟動時讀 `LC_TERMINU_ICON_WIDTH` 與 `LC_TERMINU_POWERLINE_WIDTH`；E5 的測試裡單獨量每一種 popup 的框（popup 是 lib 畫的） | — | lib 給的；有 ssh 的 app 連線時自己送 `LC_TERMINU_*`。E5 的測試只測自己的整個畫面：自己的版面加上自己會開的 popup，照 E5 的寬度與尺寸，每一列都滿寬 |
| E6 發布 | 打 git tag（Go module），沒有 binary；L4 的畫面測試要有 | 不發布；L4 的畫面測試要有 | 照 E6（畫面測試照上一列的分工）；家族的範本與 repo 的檢查在 [terminu-harness](https://github.com/vulcanshen/terminu-harness) |
| S5 splash 畫的 icon | splash 的機制；建立 splash 時，下面的字有一列超過 50 欄就回錯誤 | tdp 的 `principle/icon.svg`（terminu 的 mark） | 自己的 `docs/icon.svg` |
| E7 README | lib 自己的結構：安裝是 `go get`、快速開始是程式碼、寫明範例 app 怎麼跑 | 不另外寫 | 照 E7 |
| 做不到或不好用的規定（Rules 開頭的「放行」） | **lib 不自己另做**：回報成 tdp 的缺漏，規範改了 lib 才跟著改 | — | 回報成 tdp 的缺漏 |

這裡只寫「用 lib 時」的分工，用不用 lib 由 app 決定。app 改用 lib 時，畫面與 E5 的程式可以刪掉，E1、E3、E6、E7 與自己的 icon 留著。

## 實作哪一版 terminu design

terminu-go 實作 **tdp v1.3.7**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
