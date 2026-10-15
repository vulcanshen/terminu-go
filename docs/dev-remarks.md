# terminu-go 開發者備忘

terminu-go 是 terminu design principle（tdp）的 Go 實作：給 app 呼叫的 lib，加上展示場 `examples/showcase`。遵循 terminu design
principle，目前對應 tdp v0.6.8。這個 repo 的任務與規範的讀法見 [`CLAUDE.md`](../CLAUDE.md)。

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
- **上色**：自己寫 SGR（`Style.Render`，一律 24-bit），淡化（`Dim`）就是改寫每一列的 SGR。

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
- **icon 與 powerline 符號各量各的**（tdp E5）：兩種寬度各自照「app 的變數 → 家族的變數 → 探測」決定
  （`<APP>__ICON_WIDTH`／`TERMINU__ICON_WIDTH`、`<APP>__POWERLINE_WIDTH`／`TERMINU__POWERLINE_WIDTH`）；探測時印一個 file icon、
  一個 powerline 符號，各問一次游標位置。有整列 cursor 的地方，右邊的留白跟箭頭一樣寬（layout/popup、layout/panel）。

## 已否決，不要重提

- **bubbletea v2 與 lipgloss v2**：見「設計決定」。
- **用 lipgloss 的 `Style` 排版、`Join*`、`Place`，或 bubbletea-overlay 疊 popup**：icon 佔兩格時框歪，而且換不了寬度函式（C1）。

## 已知的牆與未做

- **自己開 bubbletea program 的 app 沒有那一層輸出**：只拿零件用、畫面自己管的 app（例：locku 的鎖定畫面），renderer 補的「清到
  行尾」不會被拿掉。要不要把 `screenOutput` 開放出去，等第一個這樣用的 app 再決定。
- **splash**（Rules S）：機制是 `App.Splash`（`V` 打開、任意鍵關掉、不揭露、只在 panel 上、10ms 的 tick）；`NewSplash` 照 S5 畫：
  讀 app 的 `docs/icon.svg`，一格一個 `nf-fa-square`，下面六列字，揭露動畫全家族一樣。金色的字怎麼分成一個一個字：svg 裡每一個
  XML 註解開始下一個字（`<!-- K -->`），所以 svg 要照 app 名字的順序列字。`Run` 拒絕沒有 splash、或每個 panel 都把 `V` 讓出去的 app。
  展示場的 `examples/showcase/icon.svg` 是 tdp v0.5.4 `principle/icon.svg` 的複本（E8）。
- **檔案預覽**（`preview.go`）：照 input/file-picker 的「預覽」表。語法上色用 `alecthomas/chroma/v2`（MIT，catppuccin-mocha，
  只上前景色）；PDF 用 `dslipak/pdf`（BSD-3，rsc.io/pdf 的分支，能取純文字）；壓縮檔用標準庫（zip、tar、gzip、bzip2）。圖片的 data URI
  只讀畫得到的那一段，不把整個檔案（最多 1 MB）都編碼。預覽框不能捲，PDF 讀到預覽區滿了就停，不一定讀滿 20 頁。
- **家族共用的設定 key**（Rules E3）：`hide_secrets_in_preview`、`input_tab_width` 是使用者的設定，lib 不讀設定檔；app 從自己的設定檔讀，
  設到 `App.HideSecretsInPreview`、`App.InputTabWidth`。
- **PTY 只在 unix**：`terminal_other.go` 開不了 PTY，只在畫面上寫出原因。

## 偏離 tdp

（沒有。lib 不偏離 tdp（rules E8）：做不到或不好用的記成盲點，規範改了 lib 才跟著改。）

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

還沒發布。建 GitHub repo、push、打 tag 都先問 user。
