# terminu-go

**Language**: [English](README.md) · 繁體中文

**照 [terminu design](https://github.com/vulcanshen/terminu) 畫出並驅動終端機介面的 Go 函式庫。**

你登記 panel 與動作 —— 鍵、名稱、範圍（item、panel 或 global）、執行前要不要先問、做什麼。terminu-go 照這一份登記畫出
整個畫面，並處理 core key：`Space` menu、`?` 的 key reference、每個 panel 下框的 hint、footer、confirm、淡化沒有 focus 的
地方、`Esc` 一次關一層、popup 上再疊 popup。menu、key reference 與實際的按鍵出自同一份登記，所以不會對不上。

函式庫管不到的畫面（全螢幕的鎖定畫面、好幾個並排的終端機），照樣可以用它的零件：confirm、toast 與每一種 input 都能單獨用。

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

API 還沒定案：函式庫還在用實作檢驗規範。定案後這裡會放一個最小的 app。

想看它做得到什麼，跑展示場（用假資料展示每一個元件的範例 app）：

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

terminu-go 實作 **tdp v0.6.8**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
