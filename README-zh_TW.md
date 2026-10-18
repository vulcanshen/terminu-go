# terminu-go

**Language**: [English](README.md) · 繁體中文

**照 [terminu design](https://github.com/vulcanshen/terminu) 畫出並驅動終端機介面的 Go 函式庫。**

你登記 panel 與動作 —— 鍵、名稱、範圍（item、panel 或 global）、執行前要不要先問、做什麼。terminu-go 照這一份登記畫出
整個畫面，並處理 core key：`Space` menu、`?` 的 key reference、每個 panel 下框的 hint、footer、confirm、淡化沒有 focus 的
地方、`Esc` 一次關一層、popup 上再疊 popup。menu、key reference 與實際的按鍵出自同一份登記，所以不會對不上。

函式庫管不到的畫面（全螢幕的鎖定畫面、好幾個並排的終端機），照樣可以用它的零件：confirm、toast 與每一種 input 都能單獨用。
app 自己的 popup（例如畫格子的）也放進同一疊：框裡畫什麼、按鍵做什麼由你決定；外框、開關動畫、淡化與 `Esc` 交給函式庫，
函式庫的 confirm 與 input 也會疊在你的 popup 上面。

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
	list := &terminu.List{Columns: []terminu.Column{{Title: "Task", Flex: true}}}
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
一個畫著自己 mark 的 splash（tdp Rules S）；`icon.svg` 就是那個 mark，這個範例用的是 terminu 的副本。

想看它做得到的全部，跑展示場（用假資料展示每一個元件的範例 app）：

```sh
go run github.com/vulcanshen/terminu-go/examples/showcase@latest
```

## 設定

使用者的設定裡，有兩個是 terminu 家族每個 app 共用的（tdp Rules E3）。函式庫不讀設定檔：由 app 從自己的設定檔讀出來交給它。

| key | 作用 | 預設 | 交給 |
|---|---|---|---|
| `hide_secrets_in_preview` | 檔案預覽裡藏起私鑰與只有擁有者能讀的檔 | `false` | `App.HideSecretsInPreview` |
| `input_tab_width` | 多行文字裡按 `Tab` 插入幾個空白 | `4` | `App.InputTabWidth` |

## 實作哪一版 terminu design

terminu-go 實作 **tdp v0.7.3**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
