# terminu-go 開發者備忘

terminu-go 是 terminu design principle（tdp）的 Go 實作：給 app 呼叫的 lib，加上展示場 `examples/showcase`。遵循 terminu design
principle，目前對應 tdp v0.3.0。這個 repo 的任務與規範的讀法見 [`CLAUDE.md`](../CLAUDE.md)。

## 運作方式

lib 的程式還沒開始寫。目前只有 `proto/c1`：驗證規範盲點紀錄 C1 的拋棄式原型（lipgloss v2 的圖層合成在 icon 佔兩格時框歪不歪）。
定案後刪掉，不會變成 lib 的一部分。

C1 驗出來的、lib 會照著做的方向：

- **寬度**：所有量寬度的地方走同一個函式（tdp E5）。lipgloss v2 的排版（`Style` 的 `Width`、`Padding`、`Border`、`Align`，
  `Join*`、`Place`）與合成（`Compositor`、`Canvas`、`Layer`）都用 `x/ansi` 的寬度，Nerd Font 的 icon 一律算一格，而且沒有掛鉤可以換：
  ultraviolet 有 `WidthMethod` 介面，但 `printString` 只用它量截斷的尾巴，每一格的寬度一律照 `ansi.WcWidth` / `ansi.GraphemeWidth`。
  所以 lip gloss 只拿來上色（`Foreground`、`Background`、`Bold` 套在量好寬度的字串上），補空白、截斷、框線、並排、疊 popup 都用自己的函式。
- **畫到終端機**：bubbletea v2 的 renderer 把 `View` 的字串拆成自己的格子（icon 一樣算一格），再用游標移動（含 hard tab）只補
  改變的格子。終端機讓 icon 佔兩格時，就算字串本身是對的，含 icon 的列框線也會畫歪、更新幾次之後框線會不見（`proto/c1/tmux-check.sh`
  可以重現）。`tea.WithoutRenderer()` 加上自己逐列從第 0 欄整列寫出，在同一個環境裡是對的。採哪一種：待定，見規範盲點紀錄 G22。

## 設計決定

- **寬度與疊 popup 不用 lipgloss v2，自己做**（C1）：理由見上面「運作方式」。lip gloss 只負責顏色與字型屬性。

## 已否決，不要重提

- **用 lipgloss v2 的 `Compositor` / `Canvas` 疊 popup**：icon 佔兩格時框歪，而且換不了寬度函式（C1）。

## 已知的牆與未做

- bubbletea v2 的 renderer 跟 icon 佔兩格的終端機對不上（見「運作方式」）。
- lipgloss v2 的坑（原型踩到的）：`Canvas.Compose(layer)` 不看 layer 的 `X`/`Y`，而且先清掉整塊；要定位得包進 `Compositor`。
  `Canvas.Render()` 會剪掉每一列結尾的空白，輸出的列不一定等於畫面寬（L4）。

## 偏離 tdp

（沒有）

## 設計文件導讀

- [`coverage.md`](coverage.md)：tdp v0.3.0 逐條的清單 —— 規範出處 → 展示場的哪裡、怎麼按到 → 測試名稱。也是展示場「所有的組合」的定義。
- 規範盲點紀錄：`~/Documents/sideproj/terminu/.local/family-fix/terminu-go/README.md`（C 待驗證、G 盲點）。程式裡的 `TODO(tdp G<n>)`
  對到那裡的編號。
- [`../CLAUDE.md`](../CLAUDE.md)：這個 repo 的任務、規範讀哪些、盲點怎麼記。

## 建置與開發

```sh
go test -race ./...
go vet ./...
gofmt -l .
```

C1 原型：

```sh
go run ./proto/c1 dump e5 80 24 menu fits          # 印一個畫面（e5 | lipgloss）
SHOWCASE__ICON_WIDTH=2 go run ./proto/c1 tui       # bubbletea v2 的 renderer
SHOWCASE__ICON_WIDTH=2 go run ./proto/c1 raw       # 自己逐列寫
proto/c1/tmux-check.sh 2 2 tui -- j j p            # 在 icon 佔兩格的 tmux 裡跑、按鍵、擷取畫面
```

`tmux-check.sh` 要有 `codepoint-widths` 選項的 tmux（在 3.7c 上跑過）。

## 發布

還沒發布。建 GitHub repo、push、打 tag 都先問 user。
