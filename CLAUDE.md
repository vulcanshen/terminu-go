# terminu-go

terminu design（tdp）的 Go 實作：一個像 bubbletea 那樣讓 app 直接呼叫的 lib，加上一個展示場（範例 app）。

- module：`github.com/vulcanshen/terminu-go`，套件名 `terminu`
- 授權：GPL-3.0
- 建在 bubbletea **v1**（`github.com/charmbracelet/bubbletea` v1.3.10）與 lipgloss v1（v1.1.0）上，跟五個 app 相同。
  原本定 v2，2026-10-08 改回 v1（紀錄的 C1、G22）：v2 的 renderer 與 lipgloss v2 的合成都處理不了 icon 佔兩格，要用 v2 就得
  自己重寫輸出層。user：v2 還不成熟，不該為了它打破既有狀態、重新造輪；等 v2 成熟再升。
- 規範：`~/Documents/sideproj/terminu`，目前實作的版本是 **tdp v1.0.2**

上面這些與下面「lib 的範圍」都已經跟 user 定案，不要重提。

## 你的任務

這個 repo 有兩個目的，順序不能顛倒：

1. **驗證規範。** 完全照 tdp 的文字寫出 lib 與展示場，找出規範沒寫、寫得模糊、前後矛盾、或做出來不好用的地方（下面叫「盲點」）。
   這個 repo 是第一個照 tdp 做的實作。找到的盲點記在紀錄裡（G 編號），一批一批補進了 tdp：哪一版補了哪些，見 tdp 的 CHANGELOG
   與紀錄的「已定」。
2. **成為家族共用的 lib。** 規範在 2026-10-13 確認（見「怎樣算規範確認」），家族的 app 都已改用這個 lib，不再各自實作規範：
   原本的五個（kbu、filu、sshu、webu、locku），加上 kamonu、ccdrawu、demou。它們用 GitHub 上打了 tag 的版本，本機再用
   `go.work` 指到這個 repo。lib 的缺漏多半經 terminu 的 session 轉來，成員也會直接回報。

## 規範是唯一的依據

### 讀什麼

- 讀 terminu **最新 tag** 的內容（現在是 v1.0.2）。main 上還沒發版的改動不算；不確定時用
  `git -C ~/Documents/sideproj/terminu diff v1.0.2 -- principle components` 看有沒有差。
- **v0.5.0 換了外觀**（配色不變）：框線、膠囊、menu 區塊標題、cursor 尾端的箭頭、標題大寫、固定的開關動畫。規範的圖現在用真的
  Nerd Font 字形。畫框、膠囊、cursor 的程式集中在少數幾個地方，不要散在各元件裡寫死 —— 外觀之後可能還會調。
- 規定只認這些檔的文字：`principle/README-zh_TW.md`、`principle/rules-zh_TW.md`、`components/` 底下全部。
  照 `components/README-zh_TW.md` 的順序讀：color → layout → dialog → input。`project/` 也是規定，但管的是 app 的 repo，
  lib 與展示場照 tdp E8 不套；`checklist/` 是從規範拆出來的條目，`docs/coverage.md` 用它的編號。
- 繁中是原文，英文（檔名沒有 `-zh_TW` 的）是翻譯；兩邊說的不一樣，也是盲點。
- **什麼是規定**：標成「例」的例子、「為什麼」、標成「實作參考（不是規定）」的段落以外，其他敘述都是規定。寫「app 可以
  自行決定」的是明文放行（tdp P5）：範圍內交給 app，旁邊寫的規定照守；沒有明文放行的地方一律照做。
- **怎麼引用**：規則寫 `tdp K4`，components 寫節編號 `tdp popup.8`（寫在每一節的標題前面），整個元件寫 `tdp form`。

### 不讀什麼

- **不看五個 app 的程式**（`~/Documents/sideproj/` 底下的 kbu、filu、sshu、webu、locku）。唯一的例外：規範裡
  「實作參考（不是規定）」段落點名的檔案，可以讀那幾個檔（例：tdp E5 點名 filu 的 `internal/ui/width.go`），同一個 app 的其他檔不看。
  純技術上的問題（例：PTY 怎麼接、怎麼問終端機游標位置）真的需要參考 app 的寫法，先問 user。
- **不看 terminu 的 `.local/`、`.checkpoints/`、`vtp/`。** 那裡有五個 app 的畫法調查、討論紀錄與舊版規範，看了就會照它們
  補上規範的空白。唯一要碰的是你自己的盲點紀錄（見下面）。
- `CHANGELOG.md` 可以用來看「哪一版改了什麼」，但規定只從上面那些檔的文字來：只有 CHANGELOG 寫到、本文沒寫的，算盲點。

**為什麼**：這個 repo 的價值在於只靠規範的文字做事。照 app 的程式或討論紀錄補上的地方，盲點就被蓋掉了，要等五個 app
改用 lib 時才會冒出來。

## 盲點：不在 lib 裡自己決定

碰到下面任何一種，**不在程式裡自己決定**，記下來：

- **沒寫**：規範沒說這種情況該怎樣。
- **模糊**：讀得出兩種意思。
- **矛盾**：兩個地方說的不一樣（含繁中與英文不一樣）。
- **方案不夠**：components 列的方案沒有一個合用（`components/README-zh_TW.md` 的「在方案裡選」）。
- **做不到或不好用**：照規範做出來不能用，或明顯難用（Principle P0：規則服務 UX）。照做，但記下來。

不算盲點、由 lib 自己決定的，是使用者看不到也按不到的事：API 怎麼設計、型別與名稱、內部結構、效能。
判斷方式：**使用者看得到或按得到的，聽規範的；app 怎麼呼叫 lib，lib 自己決定。**

### 記在哪、怎麼記

記在 `~/Documents/sideproj/terminu/.local/family-fix/terminu-go/README.md`（下面叫「紀錄」）。terminu 裡只寫這一個檔，
不改規範本身。紀錄裡有兩種編號：

- **C 編號（待驗證）**：terminu 的 session 先發現、要你在開發時實際驗證的事（例：bubbletea 怎麼收 `Alt-Esc`、
  規範指定的 glyph 實際看起來如何）。**開工前先讀一次**；做到相關的部分時照那一條的「怎麼驗」
  驗證，把結果寫回那一條（狀態改「已驗證」），搬到「已驗證」。結果顯示規範要改的，另開一條 G，兩條互相註明。
- **G 編號（盲點）**：你自己發現的盲點。編號接著已經有的往下：

```
### G<n> <一句話標題>
- 狀態：待定
- 規範：<規則編號或節編號，例 K4、popup.7>（tdp v1.0.2）
- 卡在：lib 需要知道什麼才能做下去
- 可能：A …；B …（各附一句後果）
- 暫時：跳過／暫時照 A 做（程式裡標 `TODO(tdp G<n>)`）
```

記完那一塊先跳過，繼續做別的。非往下做不可時，可以暫時選一個：程式裡標 `TODO(tdp G<n>)`，紀錄的「暫時」寫選了哪個；
定案後照定案改，拿掉標記。

### 之後怎麼走

1. 盲點累積一批、或驗完一批 C，跟 user 說；user 把它們帶到 terminu 的 session。
2. terminu 的 session 跟 user 一條一條定案、補進規範、發新版 tdp。
3. 新版發布後：
   - 紀錄裡的狀態改成「已補（tdp vX.Y.Z）」，搬到「已定」。不用改規範就定案的（例：G22），狀態寫「已定（不改規範）」。
   - 用 `git -C ~/Documents/sideproj/terminu diff <舊 tag> <新 tag> -- principle components checklist` 看完整的改動，不只看 CHANGELOG。
   - 對照 `docs/coverage.md` 每一條是否受影響，改 lib 與展示場。
   - README 與 CHANGELOG 寫的 tdp 版本跟著改。

### user 走展示場時的回饋

user 在這個 session 手動測展示場、說哪裡怪或想要什麼樣子時，先判斷是哪一種：

1. **lib 沒照規範做**（規範寫了，畫錯或做錯）：直接在這裡改，不用經過 terminu。
2. **規範本身的問題**（規範沒寫、寫得模糊，或照規範做出來 user 覺得不好用、想換做法）：不改成 user 當場說的樣子，
   在紀錄的「待定」記一條 G（user 當場說的做法寫進「可能」，註明是 user 說的）；照「之後怎麼走」回 terminu 定案、發版後再改。

分不出是哪一種時，當成第 2 種記下來，跟 user 說。C 的結果記在紀錄的「已驗證」；驗出不對的，照第 2 種記成 G。

## lib 的範圍

- **lib 管整個畫面與 core key。** app 登記 panel 與動作（鍵、名稱、範圍 item／panel／global、執行前要不要先問、做什麼）；
  Space menu、`?` 的 key reference、panel 下框的 hint、footer、confirm、淡化、`Esc` 一次一層、popup 堆疊，都由 lib 照登記產生。
  - 為什麼：家族過去走樣的，多半是元件之間的事（淡化、hint、key reference 的鍵）；而且 Principle P1 只有在 menu、`?`
    與實際的按鍵出自同一筆登記時，才保證得了。
- **零件也能單獨用。** 有些畫面 lib 管不到（例：全螢幕的鎖定畫面、好幾個 PTY），app 自己畫，但 confirm、toast、input 照樣用 lib 的。
- **放行的事 lib 不替 app 決定。** 規範寫「app 可以自行決定」的（Space menu 放什麼、panel 上的熱鍵、`Enter` 在這裡做什麼、
  畫面怎麼排、內容的顏色），lib 要讓 app 決定，不寫死。規範列了幾種方案的，每一種 lib 都提供，讓 app 挑。
- **API 要容得下五個 app，不只展示場。** 它們的性質如下（只用來想 API，不要去看它們的程式）：
  - locku：帶 PIN 鎖的螢幕保護程式 —— 全螢幕的動畫畫布、PIN 輸入、一列一個值的設定畫面。
  - webu：把網頁當文件讀的瀏覽器 —— app 自己排版的頁面內容、DevTools 的 tab、選取模式。
  - sshu：ssh 與 sftp 前端 —— 好幾個 PTY session、host 清單與篩選、帳密表單、遠端檔案清單。
  - filu：檔案管理員 —— 檔案清單與標記、預覽、shell PTY。
  - kbu：Kubernetes dashboard —— 很多資源清單與 tab、YAML 檢視、串流的 log、exec PTY、namespace 多選。
  - 後來的 kamonu（畫家族 mark 的編輯器，有 app 自己的 popup）、ccdrawu（Claude Code 的狀態列）、demou（家族的教學）也用這個 lib。

## 技術

- **畫面輸出用 bubbletea v1 的 renderer**：它整列重寫改變的列，icon 佔兩格也對得上（C1 驗出 v2 的 renderer 做不到，見 G22）。
  `proto/c1` 在 2026-10-12 改成 v1 重驗過：icon 佔兩格與一格的 tmux 裡框線都對齊、更新後不壞（紀錄的 C1）。
- **排版、截斷、疊 popup 都用 lib 自己的顯示寬度函式（tdp E5）**，lip gloss 只拿來上色；不用 bubbletea-overlay，也不用
  lip gloss 的 `Style` 排版（`Width`、`Border`、`Padding`）、`JoinHorizontal`／`JoinVertical`、`Place`。C1 的「e5」做法就是這個。
- **之後升 v2**：等 bubbletea v2 的 renderer 與 lipgloss v2 的排版能指定寬度函式（或正確處理 icon 佔兩格）時再評估。
  輸出那一層盡量寫成換得掉的，到時改動才集中。
- 測試：
  - 能測的規定都要有測試，測試名稱帶規則編號或元件名（例：`TestK4EscOneLayer`、`TestToastReplace`），從 `docs/coverage.md` 查得到。
  - L4 畫面測試：多種終端機尺寸下，每一列剛好等於終端機寬度；再跑一次 icon 佔兩格（E5）；每一種 popup 各開一次，
    含比畫面寬、比畫面高、兩邊都大三種情況。
  - 發版前跑 `go test -race ./...`、`go vet ./...`、`gofmt -l .`。

## 展示場

放在 `examples/showcase`，app 名叫 `showcase`（環境變數 `SHOWCASE__…`）。資料是假的，但排得像一個真的 app。
另外有 `examples/minimal`：README 的快速開始，最短的一個 app。

**要展示「所有的組合」**（user 定）：panel、input、dialog 全部展示出來，用不同的畫面（screen chip）呈現。現在有
`Panels`、`Inputs`、`Dialogs`、`Terminal`、`Kinds` 五個畫面（起點是前三個，放不下時再加），每個畫面的主題要一句話說得出來。「所有的組合」是：

1. 每一個元件都出現。
2. 規範列了幾種方案的，每一種方案都出現。
3. 規範寫到的每一種狀態都切得出來：有 focus 與失焦、停用、錯誤、空的、loading、模式、窄寬度、icon 佔兩格。
   切換用展示場自己的動作，放在 Space menu 裡（這也順便測了 M 章）。
4. 規範寫到的每一種疊法都走得到（例：表單 → select → finder 三層；popup 上跳 toast；Space menu 開 confirm；任何一層按 `?`），
   每一層都看得到淡化只做一次、`Esc` 一次關一層。
5. 畫面層級的東西每一種都出現：footer 的每一種樣子、statusbar、畫面 chip 列、splash（`V`）。

**`docs/coverage.md` 就是這份清單的定義。** 從規範逐條整理，每一條寫「規範出處 → 在展示場的哪裡、怎麼按到 → 測試名稱」；
看不到畫面的規定（例：E 章的命令列與環境變數）寫怎麼驗。**先整理清單、再寫程式**；整理時找到的盲點就先記下來。

## 怎樣算「規範確認」

三件事都做到，才算規範確認，五個 app 才開始改用 lib（2026-10-13 已確認：user 在 tdp v0.5.9 走過展示場）：

1. `docs/coverage.md` 每一條都有位置，在展示場都按得到（或寫明怎麼驗）。
2. 紀錄裡的 C 都驗完了；G 都已定案、補進 tdp 並發版，lib 也跟上了。
3. user 親自走過展示場，確認沒問題。

## lib 與展示場適用哪些規定（tdp E8）

原本的盲點 G1，已補進 tdp v0.4.0 的 **E8**：一張表寫 lib、展示場、用 lib 的 app 各負責哪些規定。照 E8 做。要點：
lib 做到畫面的規定與 E5；展示場不發布，照 E1、E2 與 L4 的畫面測試做；**lib 不自己另做**：做不到或不好用的一律回報成 tdp 的
缺漏（記成盲點），規範改了 lib 才跟著改。

## repo 的基本檔案與發版

1. `go.mod`（`github.com/vulcanshen/terminu-go`）、`LICENSE`（GPL-3.0 全文）。
2. `README.md` 與 `README-zh_TW.md`，兩份內容對齊：一句話定位、它能做什麼、怎麼裝（`go get`）、快速開始、實作哪一版 tdp、
   License。只寫使用 lib 的開發者需要知道的；運作方式與設計理由放 `docs/dev-remarks.md`（tdp 不管開發者文件，這份是這個
   repo 自己的備忘）。
3. `CHANGELOG.md`：每一版寫它對應的 tdp 版本。
4. push、打 tag 先問 user；terminu 的 session 定的版號算 user 同意，直接打 tag、push（user 2026-10-20）。發版後通知所有開著的
   成員 session（webu 除外，user 自己更新），由成員自己判斷要不要跟進（user 2026-10-24、10-25）。

## 跟 user 合作的方式

- 一次問一題，附上你的建議與理由。
- 先講操作（使用者會按什麼、看到什麼），再寫程式。
- 提案前先查規範怎麼說。
- user 看不懂時，用一個具體的例子重講。
- 只改這個 repo 與自己的盲點紀錄；不改 terminu 的規範，也不改家族成員的 app。
