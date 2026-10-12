# terminu-go

**Language**: [English](README.md) · 繁體中文

**照 [terminu design](https://github.com/vulcanshen/terminu) 畫出並驅動終端機介面的 Go 函式庫。**

你登記 panel 與動作 —— 鍵、名稱、範圍（item、panel 或 global）、執行前要不要先問、做什麼。terminu-go 照這一份登記畫出
整個畫面，並處理 core key：`Space` menu、`?` 的 key reference、每個 panel 下框的 hint、footer、confirm、淡化沒有 focus 的
地方、`Esc` 一次關一層、popup 上再疊 popup。menu、key reference 與實際的按鍵出自同一份登記，所以不會對不上。

函式庫管不到的畫面（全螢幕的鎖定畫面、好幾個並排的終端機），照樣可以用它的零件：confirm、toast 與每一種 input 都能單獨用。

建在 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 與 [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) 上。

## 安裝

```sh
go get github.com/vulcanshen/terminu-go
```

需要 Go 1.26 以上。用它做的 app，對使用者終端機的需求跟 terminu design 一樣：[Nerd Font](https://www.nerdfonts.com/) 與
24-bit 色（truecolor）。

## 快速開始

API 還沒定案：函式庫還在用實作檢驗規範。定案後這裡會放一個最小的 app。

## 實作哪一版 terminu design

terminu-go 實作 **tdp v0.3.0**。每一版對應的 tdp 版本寫在 [CHANGELOG](CHANGELOG.md)。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [開發者備忘](docs/dev-remarks.md)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
