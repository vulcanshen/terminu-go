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
  `FormField.Styled`，`App.Status` 也回 span。在 `Canvas` 或自己的 popup 裡畫外面來的字時，用 `Escape` 照同一套畫。
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

在任何 panel 上按 `!` 打開 App Log：app 做過的事與出問題的事，最新的在最下面，只放在記憶體裡。每一個 toast（inspect toast
除外，它只跟著游標）與 error popup 都會自己記進去；其他的用 `app.Log(kind, text)` 記（動作的結果、app 在背景做的事），
任何 goroutine 都能叫；寫法跟 toast 一樣，大寫開頭。有沒看過的錯誤或警告時，footer 的 `!:log` 會寫出有幾個。

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

## app 自己的模式

`panel.EnterMode(&terminu.Mode{...})` 讓 panel 進入 app 自己的模式（tdp K11）：只有模式的 `Keys` 與 core key 作用，`Esc` 離開。
`ModeKey` 只收 `Keys` 列出來的鍵，lib 不會替你補：方向鍵與 `PgUp|PgDn` 照 tdp K12 自己列進去；key reference 與 footer
寫的是 `Key`，只寫字母：

```go
{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
```

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

## lib 與 app 的分工

tdp E8 對任何共用的 lib 只留三點：用 lib 的 app 不另外做 lib 做到的、畫面測試照誰畫的分、lib 不自己另做。這個 lib 跟用它的 app
怎麼分工，照這張表：

| 規定 | lib | 範例 app | 用 lib 的 app |
|---|---|---|---|
| 畫面的規定（K、M、L、F、X、T、S1–S4、components） | 做到 | 全部展示出來 | lib 給的，自己不另外做 |
| E1 `version`、`help` | — | 要 | 自己做 |
| E2 環境變數的命名 | lib 自己讀的變數掛在 app 的名字底下（例：`<APP>__ICON_WIDTH`） | 照 E2（例：`SHOWCASE__…`） | 自己的變數照 E2 |
| E3 設定與資料的位置 | — | 不存東西，不適用 | 自己做 |
| E4 需求 | README 寫明 | — | README 寫明 |
| E5 icon 的實際寬度 | 做到：探測（icon 與 powerline 符號）、顯示寬度函式、替子程序設 `TERMINU__ICON_WIDTH` 與 `TERMINU__POWERLINE_WIDTH`、啟動時讀 `LC_TERMINU_ICON_WIDTH` 與 `LC_TERMINU_POWERLINE_WIDTH`；E5 的測試裡單獨量每一種 popup 的框（popup 是 lib 畫的） | — | lib 給的；有 ssh 的 app 連線時自己送 `LC_TERMINU_*`。E5 的測試只測自己的整個畫面：自己的版面加上自己會開的 popup，照 E5 的寬度與尺寸，每一列都滿寬 |
| E6 發布 | 打 git tag（Go module），沒有 binary；L4 的畫面測試要有 | 不發布；L4 的畫面測試要有 | 照 E6（畫面測試照上一列的分工）；家族的範本與 repo 的檢查在 [terminu-harness](https://github.com/vulcanshen/terminu-harness) |
| S5 splash 畫的 icon | splash 的機制；建立 splash 時，下面的字有一列超過 50 欄就回錯誤 | tdp 的 `principle/icon.svg`（terminu 的 mark） | 自己的 `docs/icon.svg` |
| E7 README | lib 自己的結構：安裝是 `go get`、快速開始是程式碼、寫明範例 app 怎麼跑 | 不另外寫 | 照 E7 |
| 做不到或不好用的規定（Rules 開頭的「放行」） | **lib 不自己另做**：回報成 tdp 的缺漏，規範改了 lib 才跟著改 | — | 回報成 tdp 的缺漏 |

這裡只寫「用 lib 時」的分工，用不用 lib 由 app 決定。app 改用 lib 時，畫面與 E5 的程式可以刪掉，E1、E3、E6、E7 與自己的 icon 留著。

## 實作哪一版 terminu design

terminu-go 實作 **tdp v1.2.2**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
