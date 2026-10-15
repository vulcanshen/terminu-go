# 手動走一遍展示場

規範確認的第三件事是 user 親自走過展示場（`CLAUDE.md`「怎樣算規範確認」）。這份是起點：怎麼開、每個畫面看什麼、紀錄裡的 C2–C4 怎麼驗。
每一條規定在哪裡按得到，查 [`coverage.md`](coverage.md)。

## 開起來

先在平常的終端機建好執行檔（放在 repo 外面，兩種環境都跑同一個）：

```sh
cd ~/Documents/sideproj/terminu-go && go build -o /tmp/showcase ./examples/showcase
```

- **tmux 裡**：平常的 Ghostty 視窗直接跑 `/tmp/showcase`。
- **不進 tmux 的 Ghostty**：`open -na Ghostty.app --args --command="/bin/zsh -f"` 開一個視窗，在裡面跑 `/tmp/showcase`。

展示場的假檔案（假的專案與假的 `.ssh`）每次啟動時寫在 `/tmp/terminu-showcase`，看完可以整個刪掉。

開起來先看右上角的 statusbar：`icon N  powerline N` 是啟動時問終端機問到的 icon 與 powerline 符號各佔幾格（E5）。框線歪了，
多半是這兩個數字跟實際不合；可以用 `SHOWCASE__ICON_WIDTH=1`（或 `2`）、`SHOWCASE__POWERLINE_WIDTH` 蓋過去比較。statusbar 其他的字：
`[N]arrow: off`（有熱鍵的 label）、`err` 與 `warn` 是 `[2] LOG` 裡錯誤與警告的行數（Red、Peach）、`[S]tart task` 進行中是 Green。

拉大拉小視窗都試：全螢幕、半螢幕、很窄（59 欄以下一次只畫一個 panel）。

## 在每個畫面都能按的

| 鍵 | 做什麼 |
|---|---|
| `P` `I` `D` `t` `K` | 換到 Panels、Inputs、Dialogs、terminal、Kinds 畫面（terminal 的鍵是小寫的 `t`，chip 寫成 `[t]ERMINAL`） |
| `Space` | 這裡能做什麼（Space menu）；最下面一列是 Global operation |
| `?` | 這裡的按鍵（key reference），在任何一層都能按 |
| `Tab`、`1`–`N` | 換 panel |
| `Esc` | 一次關一層 |
| `q`、`Ctrl-C` | 離開（先問） |
| `!` | 展示場自己的 core key（Rules K1）：在每個 panel 與 popup 上都打開 app log；打字列上是字元 |

- P 的 `[1] PODS` 上按 `D` 是這個 panel 自己的 `[D]rag`，不會換到 Dialogs 畫面（panel 的熱鍵優先，Rules M4）。要去 Dialogs，
  先 `Tab` 到別的 panel 再按 `D`，或從 Global operation 選。
- Global operation 裡有展示場自己的開關：`[N]arrow layout`（不管多寬都一次畫一個 panel）、`[1] About`（數字的 global 熱鍵，
  只在這個 popup 裡按得到）、`[S]tart task`（chip 列下的進度條）、`Hide secrets in preview`。

## 五個畫面看什麼

**P — Panels**：panel 的每一種長相與狀態。
- `[1] PODS` 是一般清單：`j|k` `u|d` `gg|G` 移、`h|l` 換 tab（PODS／EVENTS）、`/` 篩選、`m` 標記、`x` 刪除（confirm）、
  `s` 排序、`go` 跳到第一個沒在跑的、`D` 拖曳模式（框與模式名變色，拿著的那一列 Yellow、標記欄畫上下箭頭；`j|k` 搬、到頭尾繞回，`Enter` 放下，`Esc` 放回）。
- `Space` 裡有切狀態的開關：`Empty`、`Loading`、`Fail`、`Disable`，與四種 toast（`Toast info`、`Toast warning`、`Toast error`、`Toast long`）。
  `Disable` 之後下框 hint 的 `m:mark` 變停用色；I 的 `[2] FILES` 選了只列按得了的，`Disable` 之後 `V:view` 不見。
- `[2] LOG` 是一直進來的串流：`Enter` 用 viewer 看全部；`F` 是 `Frozen` 模式，這個模式不暫停 `Tab`，換到別的 panel 時看模式名的細線接頭。
- `[3] SETTINGS` 是一列一個值；在它上面 `Space` → `Fail`，`Timeout` 的 label 變 Red。

**I — Inputs**：每一種 input。
- `[1] INPUTS` 每一列 `Enter` 打開它的 popup：文字、密碼、PIN、數字、select、checkbox、textarea、日期、顏色、slider、檔案。
  `Segments` 是 reorder：`D` 拿起一項、`j|k` 搬（到頭尾繞回）、`Enter` 放下，再按 `Enter` 寫回；搬過之後 `Esc` 先問要不要放棄。
  `Host` 是預覽由 app 給的 finder（沒有下面的四列資訊）。`Path` 與 `Download to` 開「選目錄」的 file-picker：清單只有目錄，
  最上面一列 `. (this directory)` 按 `Enter` 選目前的目錄。**`Download to` 打開的是你真的 `~/Downloads`**，清單與預覽的樹會列出真的檔名。
- `Identity file` 開 file-picker，起點是假的 `.ssh`。按 `/` 到打字列，打路徑與 wildcard 試試看，例：`/tmp/terminu-showcase/.ssh/**/*.pem`、
  `/tmp/terminu-showcase/project/`（假的專案：`empty` 是空目錄、`locked` 讀不到、`broken.pdf` 是斷掉的 link）。
  - **`~/` 開頭的路徑是你真的家目錄**，預覽會畫出真的檔案（包括私鑰）。要試 `~/`、又要截圖時，先在 Global operation 打開
    `Hide secrets in preview`。
- `[2] FILES`：`/` 開有預覽的 finder，每一種檔各一個（文字、圖、超過 1 MB 的圖、PDF、沒有文字的 PDF、壓縮檔、二進位、symlink、讀不到的檔）；
  `S` 是結果慢慢進來的搜尋；`V` 用 viewer 開檔案（這個 panel 把 `V` 拿去用了）。

**D — Dialogs**：每一種 dialog 與疊法。
- `[1] DIALOGS` 每一列是一個情境，`Enter` 打開：各種 menu（分組、兩步、工作清單、很長）、confirm、表單、viewer（選取模式按 `v` 兩次）、
  有 tab 的 popup（也有選取模式：平常 `h|l` 切 tab，`v` 之後 `h|l` 移游標）、loading popup、error popup（短的與長的）、toast、
  好幾層疊在一起。每一列的說明寫了要看什麼。
- 每一層看：只有最上層亮、底下淡化一次，`Esc` 一次關一層。

**t — terminal**：PTY。
- `[1] SHELL` 是 panel 裡的 shell：`Enter` 進去打字，`Alt-Esc` 出來（先問）。在外面按 `?`，最後一段 `TERMINAL` 寫 shell 裡的鍵。
- `[2] SESSIONS` 每一列 `Enter` 開一個 terminal popup：`shell`、`keys`（`cat -v`，看每個鍵送了什麼）、`top`、`vi`。
- popup 裡 `Alt-t` 藏起來；`Alt-b` 讓展示場卡 2 秒（C2 用），卡完會跳一個 toast。

**K — Kinds**：清單分組與明細。
- `[1] KINDS` 是分組的清單（WORKLOADS、NETWORK、CONFIG、STORAGE、CLUSTER）：`j|k` 會跳過分組標題，`gg|G` 到第一個、最後一個種類；
  `/` 篩選，被篩掉的組連標題一起不見。
- `[2]` 是 `[1]` cursor 那一種的明細：cursor 一動就換，label 寫那一種的名字；focus 留在 `[1]`，要到 `[2]` 按 `Tab` 或 `2`。
- 在 namespaced 的種類上按 `Enter` 也會進到 `[2]`（Rules K3 唯一可以用 `Enter` 換 panel 的地方），`Short name` 可以改；
  cluster 層級的種類（`persistentvolumes`、`storageclasses`、`nodes`、`namespaces`）按 `Enter` 不進去，只跳一個 toast。
- 把終端機拉矮，捲到一組的第一個種類時，那一組的標題也留在畫面裡。

**splash**：在 panel 上按 `V`（I 的 `[2] FILES` 除外），任意鍵關掉。畫的是 tdp 的 mark（藍色 U 框裡一個金色的 T），底 → T → U 框一段一段揭露，最後是下面的字。

## C2：`Esc` 與 `Alt-Esc`

在 **tmux 裡**與**不進 tmux 的 Ghostty** 各做一次。`t` → `2` → 移到 `keys` → `Enter`，畫面上會印出每個鍵送給程式的 byte
（`Esc` 印成 `^[`）。

1. 單按一次 `Esc`：應該印出 `^[`，留在 popup 裡。
2. `Alt-Esc`（Option + Esc）：應該跳出 confirm（`Exit keys?`）；`Esc` 回到 PTY，`Enter` 離開。
   - Option + Esc 什麼都沒發生、或印出別的字時，多半是 Ghostty 的 `macos-option-as-alt` 沒開。照實跟我說就好，不用為了測試改設定。
3. 很快連按兩次 `Esc`：記下是印出 `^[^[`，還是跳出 confirm。
4. 先按 `Alt-b`（展示場卡 2 秒），卡住的時候連按兩次 `Esc`：記下是印出 `^[^[`，還是跳出 confirm。
5. 打字列上的 `Ctrl-`／`Alt-` 組合不打字：I → `[1] INPUTS` 的 `Name` → `Enter`，按 `Ctrl-t`、`Alt-x`、`Ctrl-g`，框裡不應該多出字。

每一種記下「在哪個環境、按了什麼、看到什麼」，跟我說。

## C3：排序 glyph 的方向

P → `[1] PODS`：按 `s`（Sort by），選 `Name`，再選 `Then by age`：照名字升冪、再照 age 降冪，表頭 `NAME` 旁邊是升冪的 glyph
（U+F160）、`AGE` 旁邊是降冪的 glyph（U+F161），兩個同時看得到。

看兩個 glyph 各自「看起來」是由小到大還是由大到小，跟規範的名稱（升冪、降冪）對不對得上。截一張表頭的圖。

## C4：每一個 glyph 畫出來看一次

[`coverage.md`](coverage.md) 最後的「glyph（C4 的清單）」一節列了規範指定的每一個 glyph，以及在展示場的哪裡出現。每一個看三件事：

1. **前進幾格**：glyph 後面的字有沒有被吃掉或多空一格；它所在那一列的右框有沒有對齊。
2. **清不清楚**：尤其 outline 的（`nf-md-checkbox_blank_outline`、`nf-md-shield_alert` 等）線會不會太細。
3. **意思對不對**：例：Warning 的 glyph 看起來像警告嗎。

截圖：`Cmd-Ctrl-Shift-4` 框選後存進剪貼簿，直接在這裡 `Ctrl-V` 貼上；或 `Cmd-Shift-4` 再按空白鍵點視窗，存成檔案後把路徑給我。
截圖前確認畫面上沒有真的私鑰（見上面 I 的 `~/`）。有問題的 glyph 說一聲是哪一個、哪裡怪就好。

## 看到怪的地方

直接跟我說哪裡怪、想要什麼樣子。我會先分兩種：

- **lib 沒照規範做**（規範寫了，畫錯或做錯）：當場改。
- **規範本身的問題**（規範沒寫、寫得模糊，或照規範做出來不好用）：不當場改成你說的樣子，記成紀錄「待定」裡的一條 G，
  你說的做法寫進「可能」；之後帶回 terminu 定案、發版，我再照新版改。

分不出是哪一種時，當成第二種記下來，並跟你說。
