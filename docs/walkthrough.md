# 手動走一遍展示場

規範確認的第三件事是 user 親自走過展示場。這份是起點：怎麼開、每個畫面看什麼、要在真的終端機上看過才知道的三件事怎麼驗。
每一條規定在哪裡按得到，查 [`coverage.md`](coverage.md)。

## 開起來

先在平常的終端機、你 clone 的 terminu-go 目錄裡建好執行檔（放在 repo 外面，兩種環境都跑同一個）：

```sh
go build -o /tmp/showcase ./examples/showcase
```

- **tmux 裡**：平常的 Ghostty 視窗直接跑 `/tmp/showcase`。
  也可以在 repo 裡打 `make showcase`，用目前的程式碼建好執行檔再打開，一步就好。
- **不進 tmux 的 Ghostty**：`open -na Ghostty.app --args --command="/bin/zsh -f"` 開一個視窗，在裡面跑 `/tmp/showcase`。

展示場的假檔案（假的專案與假的 `.ssh`）每次啟動時寫在 `/tmp/terminu-showcase`，看完可以整個刪掉。

開起來先看右上角的 statusbar：`icon N  powerline N` 是啟動時問終端機問到的 icon 與 powerline 符號各佔幾格（E5）。框線歪了，
多半是這兩個數字跟實際不合；可以用 `SHOWCASE__ICON_WIDTH=1`（或 `2`）、`SHOWCASE__POWERLINE_WIDTH` 蓋過去比較。statusbar 其他的字：
`[N]arrow: off`（有熱鍵的 label）、`err` 與 `warn` 是 `[2] LOG` 裡錯誤與警告的行數（Red、Peach）、`[S]tart task` 進行中是 Green。

拉大拉小視窗都試：全螢幕、半螢幕、很窄（59 欄以下一次只畫一個 panel）。

## 命令列

不打開畫面、直接印出來的指令（E1、E3）：結果在 stdout，說的話在 stderr，一行一件、開頭 `showcase: `。每個後面都 `echo $?` 看一下。

- `/tmp/showcase version > /tmp/v`：版號在 `/tmp/v` 裡，畫面上什麼都沒有；exit 0。
- `/tmp/showcase iconwidth`：兩行 `icon N (來源)`、`powerline N (來源)`；`SHOWCASE__ICON_WIDTH=2 /tmp/showcase iconwidth` 的來源寫
  那個變數（E1-3）。
- `/tmp/showcase settings | cat`：pipe 只收到設定；終端機上四行，跟 D「Values that do not fit」的 toast 與 App Log 同一句話
  （toast 那一行、兩筆 WARN），再加 `No longer used`（E3-40）；exit 0。
- `/tmp/showcase settings broken`：`… can't be read: …` 與一行 `Open showcase to move it aside and start fresh`，什麼檔都不動；exit 1。
- `/tmp/showcase x`：`showcase: Unknown command "x"`、空一行、用法；exit 2。
- `/tmp/showcase update`：展示場不發布，永遠是從原始碼編的：stderr 一行 `showcase: This showcase is built from source: update it there
  (…)`，什麼都不查；exit 1（E1-10）。其他安裝方式（Homebrew、install script、Scoop、已經最新、沒有權限）由 lib 的測試對著假的 release 驗。

## 在每個畫面都能按的

| 鍵 | 做什麼 |
|---|---|
| `P` `I` `D` `t` `K` | 換到 Panels、Inputs、Dialogs、terminal、Kinds 畫面（terminal 的鍵是小寫的 `t`，chip 寫成 `[t]ERMINAL`） |
| `Space` | 這裡能做什麼（Space menu）；最下面一列是 Global operation |
| `?` | 這裡的按鍵（key reference），在任何一層都能按 |
| `Tab`、`1`–`N` | 換 panel |
| `Esc` | 一次關一層 |
| `q`、`Ctrl-C` | 離開（先問） |
| `!` | App Log（每個 app 都有）：toast、error popup 與展示場記下的事；footer 有沒看過的錯誤時寫成 `!:N errors`；模式裡與打字列上不作用 |
| `%` | 展示場自己的 core key（Rules K1）：在每個 panel 與 popup 上都打開展示場現在的狀態；打字列上是字元 |

- P 的 `[1] PODS` 上按 `D` 是這個 panel 自己的 `[D]iff`，不會換到 Dialogs 畫面（panel 的熱鍵優先，Rules M4）。要去 Dialogs，
  先 `Tab` 到別的 panel 再按 `D`，或從 Global operation 選。
- Global operation 裡有展示場自己的開關：`[N]arrow layout`（不管多寬都一次畫一個 panel）、`[1] About`（數字的 global 熱鍵，
  只在這個 popup 裡按得到）、`[S]tart task`（chip 列下的進度條）、`Hide secrets in preview`。

## 五個畫面看什麼

**P — Panels**：panel 的每一種長相與狀態。
- `[1] PODS` 是一般清單：`j|k` `u|d` `gg|G` 移、`h|l` 換 tab（PODS／EVENTS）、`/` 篩選、`m` 標記、`x` 刪除（confirm）、
  `s` 排序、`go` 跳到第一個沒在跑的、`D` Diff 模式（`[D]iff`，K13）、`X` 刪掉標記的、`r` 拖曳模式（`D[r]ag`，因為 `D` 給了 Diff；框與模式名變色，拿著的那一列 Yellow、標記欄畫上下箭頭；`j|k` 搬、到頭尾繞回，`Enter` 放下，`Esc` 放回）。
  寬度有剩時（panel-184）：把終端機拉到 240 欄左右，Space menu 的 `Room left over` 一次換一種：放最後、`NAME` 填滿（`AGE` 靠到右邊）、
  平均、加大欄距，每換一次跳一個 Info toast 說是哪一種。
  `EVENTS` tab 是占好幾列的項目（panel-181～183）：事件的訊息在 REASON 底下，`BackOff` 兩列、`Pulled` 一列、`Started` 只有一列；
  cursor 只畫在第一列、箭頭在那一列，下面幾列保留字的顏色，背景從左邊 `#2f313c` 開始很快變淡、到右邊框變成終端機自己的底色，看不出塗到哪裡；按 `2` 讓 PODS 失焦、或按 `?` 疊一層 popup，起點跟著變暗，右端照樣看不出界線；`j|k` 一次一項；按 `/` 打 `worker-2`，找到 `web-a8k2`。
  項目的附註（panel.18）：沒在跑的 pod（`coredns-7w9z` 的 CrashLoopBackOff、Pending 的）有附註，cursor 停上去，那一列正下方出現
  一個小框 `┌ 󰋽 i ┐` 蓋在下一列上，移開就消失；按 `i` 從那裡往下展開一個 `NOTE` viewer 寫原因，可以 `/` 搜尋、`v` 選取，`Esc`
  收回成頭。`G` 到最後一個 pod（`traefik-4b`）：頭在那一列上面（`└ 󰋽 i ┘`），`i` 往上開。Space menu 裡有 `[i] Note`。
- `Space` 裡有切狀態的開關：`Empty`、`Loading`、`Fail`、`Disable`，與五種 toast（`Toast success`、`Toast info`、`Toast warning`、`Toast error`、`Toast long`）。
  `Disable` 之後下框 hint 的 `m:mark` 變停用色；I 的 `[2] FILES` 選了只列按得了的，`Disable` 之後 `V:view` 不見。
- `[2] LOG` 是一直進來的串流：在結尾時跟著往下，往上捲就停在那裡；`Enter` 用 viewer 看全部。`Space` 裡的 `Status` 把最後一行進來的時間
  放到框上，每選一次換一個角；`Status trouble` 讓它變 Peach、Red，再變回來。
- `[3] SETTINGS` 是一列一個值；在它上面 `Space` → `Fail`，`Timeout` 的 label 變 Red。`Space` → `Profile gone` 把使用者自己的 profile `work`
  拿掉，像手動刪掉後再啟動：跳 Warning toast（`!` 裡是一筆 WARN），`Profile` 照樣寫 `work`、label 變 Red；再選一次，它回來了，又能用。
  `Profile` 上按 `Enter` 開 select（使用者自己的資料一律開 popup）。

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
  `S` 是結果慢慢進來的搜尋；`V` 用 viewer 開檔案（這個 panel 把 `V` 拿去用了）。`h|l` 換到 `LINES` tab：`main.go` 一行一列，
  `Text` 欄是一行一行的文字，`fmt.Println` 那行的 Tab 畫成空白的縮排，不是紅色的 `\t`。
  - 同一項的兩個版本（note.8）：在 `FILES` tab 的檔上按 `H`，跟它最後一次 commit 比（`main.go`、`README.md` 有改，其他檔沒變，
    跳 Info toast）；按 `K` 進到最後一次 commit 改了的檔：改名的 `~ docs/NOTES.md → docs/notes.md`、新增、刪掉、不是文字的圖
    （`Enter` 停用）、裡面還有東西的 `themes/`（再進一層）、路徑很長的 `installation/docker-compose/production/env.example`；
    Diff 的標題只寫 `DIFF`、後面接兩個 tab，兩邊的名字在第一列（Lavender，緊貼上框、上面沒有空白列，捲動時不動）；在那個長路徑上按 `Enter`，100 欄時
    在 `↔` 前面折成兩列，縮到 60 欄時再在 `/` 後面折，不用 `…` 截（note-107）；Diff 的寬度是終端機寬減 2（F7-17）。
    `Esc` 一層一層退回原本的檔案清單。在 `themes/` 裡時到 `[1] INPUTS`
    按 `M`（Show main.go）：`[2]` 先離開 commit 的每一層，回到檔案清單、cursor 在 `main.go`。
  - 比兩個資料夾（panel.9）：`FOLDERS` tab 進 `releases/`，在 `v1/` 按 `D`、移到 `v2/` 按 `Enter`：位置列 `releases/v1 ↔ releases/v2`，
    只列兩邊不一樣的項目（`+ NEWS.md`、`~ conf/`、`~ logo.png`、`~ main.go`、`- old-notes.txt`，一樣的 `README.md`、`docs/` 不列）；
    `conf/` 再進一層；`Esc` 一層一層退回，最後回到 Diff 模式、anchor 還在。
  - 位置列的狀態與工作（panel.10）：`Space` 選 `Merge half done`，位置後面接 Lavender 的 `Merging: feature/login`、開頭換成 merge 的 icon；再選 `Push`，後面接 Green 的 `Pushing…`、icon 換成轉著的 Green loading icon，三秒後換回 merge 的 icon。
- `[3] CHANGES` 跟著 `[2]` 的 cursor（note.8 的 In a panel）：cursor 上那個檔跟它最後一次 commit 的差異，畫法跟 UNIFIED 一樣（只有改的段落、
  兩欄行號、`-` Red、`+` Green），不是 git 印的 patch；位置列是那個檔。在 `main.go` 上看得到改的那一行，按 `3` 再按 `Enter` 打開 Diff；
  沒改的檔寫 `No changes since 5d6e7f8`（`Enter` 停用），圖寫 `Not a text file`。在 `FOLDERS` tab 的 `releases/` 裡移到 `v2/`
  （最後一次 commit 之後新增的資料夾）：`[3]` 列出它有差異的項目、每一項帶標記，全部是 Green 的 `+`（note-106）；按 `3` 再按 `Enter`
  進到這份清單；`[2]` 的 cursor 一動，`[3]` 就離開清單、換成新的那一列。
- Diff 模式放在明細裡（panel-185、panel-186）：在 `releases/` 裡按 `Space` 選 `Diff in [3]`，在 `v1/` 按 `D`：`[2]` 與 `[3]`
  兩個框都變黃、都寫 `Diff`，粗線還在 `[2]`；`[3]` 寫 `Move to another file or folder to diff with releases/v1`。按 `j` 到 `v2/`：
  不用按鍵，`[3]` 就列出 `releases/v1 ↔ releases/v2` 有差異的項目。按 `D`（或 `Enter`）focus 到 `[3]`，footer 寫 `Esc:back`；
  在 `conf/` 按 `Enter` 進去、在 `app.toml` 按 `Enter` 打開 Diff。之後一直按 `Esc`：視窗 → `conf/` → `[3]` → `[2]`（還在模式裡、
  anchor 還在）→ 離開模式，`[3]` 回到 cursor 那一列跟最後一次 commit 的差異。在模式裡 `Tab`、`2`、`3` 在 `[2]` 與 `[3]` 之間移動，
  `1` 跳 `Press [Esc] to leave Diff first`。在專案最上層（`FOLDERS` tab 一開始的那層）也試：在 `assets/` 按 `D`，移到 `main.go`：
  `Can't diff main.go with assets`；移到 `locked/`：`[3]` 中間是 Red 的 `locked: …permission denied`，不開 popup（panel-187）。
  再選一次 `Diff in [3]` 回到原本的做法。
- 篩選遇到 app 自己的動作（panel-180）：在 `releases/` 裡按 `/` 打 `v`、`Enter` 回清單，用 `D` 比 `v1/` 與 `v2/`：進去之前篩選先清掉，
  `Esc` 退回來時是整份清單、cursor 在原本那一列。

**D — Dialogs**：每一種 dialog 與疊法。
- `[1] DIALOGS` 每一列是一個情境，`Enter` 打開：各種 menu（分組、兩步、工作清單、很長）、confirm、表單、viewer（選取模式按 `v` 兩次）、
  有 tab 的 popup（也有選取模式：平常 `h|l` 切 tab，`v` 之後 `h|l` 移游標）、loading popup、error popup（短的與長的）、toast、
  好幾層疊在一起。每一列的說明寫了要看什麼。
- 「Viewer: a watched file」是一個會變的檔案，每 3 秒重新讀一次：變長、變短、同樣長但改了幾行、跟上一次一樣、讀不到兩輪（Error toast
  只跳第一輪）、換一種讀不到（再跳一次）、又讀得到，再從頭。捲到結尾時跟著新的結尾；按 `/` 找 `needle`，命中跟著新內容走；按 `v`
  兩次選一段，選到的行不見時跳 Warning；選的時候按 `Space` 或 `?` 開著，就改疊一個 Peach 的 note，`Enter` 或 `Esc` 連同 menu 一起關。
  關掉後按 `!`（note-104、note-105）：每一次變了一筆 INFO（`Changed outside the app: …`），一樣的那幾輪沒有；每一種讀不到一筆 ERR；
  又讀得到一筆 INFO，沒有 toast。
- 「Values that do not fit」：一個有兩個用不了的值的假設定檔，複製成 `config.yaml.bak`、用預設值重寫，跳一個 Warning toast；按 `!`
  看每個值各一筆 WARN。「…, no copy」是同樣的檔放在不能寫的資料夾：沒有 `.bak`、檔案不動，跳 Error toast。
- 「Several at one start」：同樣的檔，再加上一個不在的 profile：一件一個 Warning toast，照順序疊起來（E3-37）；按 `!` 看一件一筆，
  值的 WARN 緊接在它那一件後面。
- toast 往上疊（toast.2、toast.3）：連選幾次「Toast」與別的會跳 toast 的動作，新的出現在最下面、把舊的往上推，最多三個，第四個來時
  最舊的走；同一句話連跳三次，只有一個、標題寫 `INFO ×3`；按 `Esc` 一次收一個，從最上面（最舊的）開始，`Esc:close` 只畫在它上面。
- 「Toast: a colour code in a name」：名字裡夾著 `\x1b[31m`（像資料檔裡手改過的名稱）：toast 用 Red 寫出 `\x1b[31m`、後面的 `X` 沒變紅；
  按 `!`，App Log 那一筆也一樣，照 toast 的一列畫（note-108）。
- 「Success report」：toast 放不下的成功報告開在 viewer 裡，`Enter` 也能關；按 `!` 看它以 OK 記進 App Log。
- 「Form: with actions」有好幾個按鈕：`Tab` 到動作區先停在右下角的 `Connect`，`j|k` 在按鈕之間走。`Test` 只要 Host：空著按會被擋，
  填了再按要等一下（標題後面轉 loading icon）才跳 toast，Host 打 `bad-1` 會失敗、開 error popup，關掉後再按 `Enter` 就是重試；
  `Scan` 開一個 note，還把 Port 寫成 22，之後 `Esc` 會先問要不要放棄。`Test` 還在跑時，下框的 hint 沒有 `Enter`，再按也不會
  再跑一次；這時按 `Esc` 關掉表單，結果就不會跳出來。`Connect` 也要等一下，連上才關掉表單。`Export hosts.yaml` 上的 hint 是
  `Enter:export hosts.yaml`：只有動詞小寫，檔名照原樣。
- 「Popup: the app’s own」是展示場自己畫的 popup（`DRAW: icon`）：框裡的格子與按鍵是展示場的，框、動畫、淡化是 lib 的。`hjkl` 移、
  `Enter` 或 `Space` 畫一格，`Tab` 到 `Save`、再按回格子；`q` 不作用。畫過一格之後 `Esc` 先問要不要放棄，`Ctrl-C` 的離開 confirm
  多一句 Peach 的 `Changes to icon are not saved.`；`?` 只列它自己的鍵。
- 「Move aside」兩列是讀不了自己的檔時的 confirm：`a file` 是設定檔，`Enter` 移到 `.bak`、像第一次打開一樣用預設值（展示場第一次打開不建設定檔，所以不會多一個新的）；`a record` 是一筆一個檔的資料，
  問句寫 `go on without it`，`Enter` 只把壞的那一筆移開、讀其他的。兩個都是 `Esc` 直接離開展示場。
- 每一層看：只有最上層亮、底下淡化一次，`Esc` 一次關一層。

**t — terminal**：PTY。
- `[1] SHELL` 是 panel 裡的 shell：`Enter` 進去打字，`Alt-Esc` 出來（先問）。在外面按 `?`，最後一段 `TERMINAL` 寫 shell 裡的鍵。
  按 `M`（`Typing mode`）是展示場自己的模式裝著這個 shell：上框右側 `Typing`、外框換模式色，`Space`、`?`、`q` 都進 shell；`Esc` 出來、不問（K11）。
- `[2] SESSIONS` 每一列 `Enter` 開一個 terminal popup：`shell`、`keys`（`cat -v`，看每個鍵送了什麼）、`top`、`vi`。
- popup 裡 `Alt-t` 藏起來；`Alt-b` 讓展示場卡 2 秒（試 `Esc` 與 `Alt-Esc` 用），卡完會跳一個 toast。

**K — Kinds**：清單分組與明細。
- `[1] KINDS` 是分組的清單（WORKLOADS、NETWORK、CONFIG、STORAGE、CLUSTER）：`j|k` 會跳過分組標題，`gg|G` 到第一個、最後一個種類；
  `/` 篩選，被篩掉的組連標題一起不見。
- `[2]` 是 `[1]` cursor 那一種的明細：cursor 一動就換，label 寫那一種的名字；focus 留在 `[1]`，要到 `[2]` 按 `Tab` 或 `2`。
- 在 namespaced 的種類上按 `Enter` 也會進到 `[2]`（Rules K3 唯一可以用 `Enter` 換 panel 的地方），`Short name` 可以改；
  cluster 層級的種類（`persistentvolumes`、`storageclasses`、`nodes`、`namespaces`）按 `Enter` 不進去，只跳一個 toast。
- 把終端機拉矮，捲到一組的第一個種類時，那一組的標題也留在畫面裡。
- `[2]` 下面兩塊是說明牌（panel.17）：`LEGEND` 說 Green 的 `●` 是什麼（常亮，focus 在哪都不淡化），`SUMMARY` 寫有幾種、開在哪一種
  （常暗）；框上只有名字、沒有號碼，footer 寫 `Tab|1–2`。一直按 `Tab`、按 `3`、`4`：focus 只在 `[1]`、`[2]` 之間；按 `Space`：兩塊都跟著
  變暗。在 `deployments` 上按 `f`，`SUMMARY` 換成 `Opens on deployments`。縮到 60 × 32 時兩塊的字都放得下。

**R — Review**：跟著帶頭 panel 換的 panel（L5-4）。
- 左邊是兩塊帶頭的：`[1] CHANGES` 與 `[2] BRANCHES`；右上是說明牌 `REPO`（沒有號碼）。
- 一開始 focus 在 `[1] CHANGES`：右邊是 `[3] DIFF`、`[4] HISTORY`，第五個位置收起、兩塊平分空間，按 `5` 沒反應。
- 按 `2`：右邊換成 `[3] COMMITS`、`[4] DESCRIPTION`、`[5] FILES`。按 `Tab` 或 `3`～`5` 進右邊，什麼都不換；再按 `1`，又是 DIFF 與
  HISTORY，每次都一樣。footer 一直寫 `Tab|1–5`。

**splash**：在 panel 上按 `V`（I 的 `[2] FILES` 除外），任意鍵關掉。畫的是 tdp 的 mark（藍色 U 框裡一個金色的 T），底 → T → U 框一段一段揭露，最後是下面的字。

## `Esc` 與 `Alt-Esc`

在 **tmux 裡**與**不進 tmux 的 Ghostty** 各做一次。`t` → `2` → 移到 `keys` → `Enter`，畫面上會印出每個鍵送給程式的 byte
（`Esc` 印成 `^[`）。

1. 單按一次 `Esc`：應該印出 `^[`，留在 popup 裡。
2. `Alt-Esc`（Option + Esc）：應該跳出 confirm（`Exit keys?`）；`Esc` 回到 PTY，`Enter` 離開。
   - Option + Esc 什麼都沒發生、或印出別的字時，多半是 Ghostty 的 `macos-option-as-alt` 沒開。照實跟我說就好，不用為了測試改設定。
3. 很快連按兩次 `Esc`：記下是印出 `^[^[`，還是跳出 confirm。
4. 先按 `Alt-b`（展示場卡 2 秒），卡住的時候連按兩次 `Esc`：記下是印出 `^[^[`，還是跳出 confirm。
5. 打字列上的 `Ctrl-`／`Alt-` 組合不打字：I → `[1] INPUTS` 的 `Name` → `Enter`，按 `Ctrl-t`、`Alt-x`、`Ctrl-g`，框裡不應該多出字。

每一種記下「在哪個環境、按了什麼、看到什麼」，跟我說。

## 排序 glyph 的方向

P → `[1] PODS`：按 `s`（Sort by），選 `Name`，再選 `Then by age`：照名字升冪、再照 age 降冪，表頭 `NAME` 旁邊是升冪的 glyph
（U+F160）、`AGE` 旁邊是降冪的 glyph（U+F161），兩個同時看得到。

看兩個 glyph 各自「看起來」是由小到大還是由大到小，跟規範的名稱（升冪、降冪）對不對得上。截一張表頭的圖。

## 每一個 glyph 畫出來看一次

[`coverage.md`](coverage.md) 最後的「glyph」一節列了規範指定的每一個 glyph，以及在展示場的哪裡出現。每一個看三件事：

1. **前進幾格**：glyph 後面的字有沒有被吃掉或多空一格；它所在那一列的右框有沒有對齊。
2. **清不清楚**：尤其 outline 的（`nf-md-checkbox_blank_outline`、`nf-md-shield_alert` 等）線會不會太細。
3. **意思對不對**：例：Warning 的 glyph 看起來像警告嗎。

截圖：`Cmd-Ctrl-Shift-4` 框選後存進剪貼簿，直接在這裡 `Ctrl-V` 貼上；或 `Cmd-Shift-4` 再按空白鍵點視窗，存成檔案後把路徑給我。
截圖前確認畫面上沒有真的私鑰（見上面 I 的 `~/`）。有問題的 glyph 說一聲是哪一個、哪裡怪就好。

## 看到怪的地方

直接跟我說哪裡怪、想要什麼樣子。我會先分兩種：

- **lib 沒照規範做**（規範寫了，畫錯或做錯）：當場改。
- **規範本身的問題**（規範沒寫、寫得模糊，或照規範做出來不好用）：不當場改成你說的樣子，記成一條盲點（G），
  你說的做法寫進「可能」；之後帶回 terminu 定案、發版，我再照新版改。

分不出是哪一種時，當成第二種記下來，並跟你說。
