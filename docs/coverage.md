# tdp v0.8.0 coverage

tdp v0.8.0 逐條整理成清單：**規範出處 → 展示場的哪裡、怎麼按到 → 測試名稱**。這份清單就是展示場「所有的組合」的定義
（[`CLAUDE.md`](../CLAUDE.md) 的「展示場」）。看不到畫面的規定寫怎麼驗。

- 「展示場」欄是位置，「測試」欄是測試名稱，都寫好了。位置或名稱改了就改這裡。
- 出處寫法：`K4` 是 `principle/rules-zh_TW.md` 的規則；`P3` 是 `principle/README-zh_TW.md`；`popup §外框` 是
  `components/layout/popup-zh_TW.md` 的那一節，其他元件照檔名（`form`、`finder`…）；`components §大寫` 是 `components/README-zh_TW.md`。
- `G<n>` 是規範盲點紀錄裡還沒定案的編號（`~/Documents/sideproj/terminu/.local/family-fix/terminu-go/README.md`），定案前照紀錄的
  「暫時」做。G1–G22 已經補進 tdp v0.4.0、G23–G29 補進 v0.5.1、G30–G46 補進 v0.5.2（G43 不改規範）、G47 補進 v0.5.3、G48–G66 補進 v0.5.4、G67–G77 補進 v0.5.5、G78–G82 補進 v0.5.6、G83 補進 v0.5.7、G84 補進 v0.5.8、G85 補進 v0.5.9、G86 補進 v0.6.1，這裡直接寫規範的條文。G87（有表頭的清單分組時表頭縮不縮）已定、不改規範：分組只用在單欄、沒有表頭的清單。v0.6.0 只刪了 S5 講畫 mark 的工具的一句，這份清單不受影響。G93 補進 v0.7.1，G94 補進 v0.7.2，G95 補進 v0.7.3。
- 「lib」表示由 lib 的登記或 API 保證，不需要展示場的某一個畫面；測試照樣要有。

## 展示場

app 名 `showcase`，五個畫面，畫面 chip 列 `[P]ANELS [I]NPUTS [D]IALOGS [T]ERMINAL [K]INDS`（多畫面的 app；切換是 global operation，
`P` `I` `D` `T` `K` 是 global 熱鍵）。每個畫面的 panel 各自從 `[1]` 編號。資料是假的，但排得像一個真的 app。

| 畫面 | 一句話 | panel |
|---|---|---|
| `[P]ANELS` | panel 的每一種樣子與狀態 | `[1] PODS`：一般清單（tab `PODS` / `EVENTS`、表頭排序、標記、cursor、40 列、截斷、panel filter、`[D]rag` 模式）；`[2] LOG`：串流內容、沒有 cursor，`Enter` 開 viewer、`[F]reeze` 是不暫停 `Tab` 的模式；`[3] SETTINGS`：一列一個值 |
| `[I]NPUTS` | 每一種 input | `[1] INPUTS`：一列一個值，每一種 input（含各方案）各一列，`Enter` 打開它的 popup（`Host` 是 app 自己給預覽的 finder）；`[2] FILES`：清單，`[/] Find` 開一般的 finder（有預覽）、`[S]low index` 是結果慢慢進來的 finder、`[V]iew` 用 viewer 開檔案 |
| `[D]IALOGS` | 每一種 dialog 與疊法 | `[1] DIALOGS`：每一列是一個情境（某一種 dialog、某一種狀態、某一種疊法），`Enter` 開它 |
| `[t]ERMINAL` | PTY 的兩種放法（畫面的鍵是小寫的 `t`） | `[1] SHELL`：PTY 放在 panel 裡；`[2] SESSIONS`：清單，`[Alt-t]erm` 開 terminal popup |
| `[K]INDS` | 清單分組與跟著 cursor 換的明細 | `[1] KINDS`：分組的一般清單（五組、16 種、panel filter）；`[2]`：`[1]` cursor 那一種的明細（一列一個值），標題 `[2] KIND: pods`，名稱照原樣 |

**展示場自己的動作**（切換狀態用，順便測了 M 章）：P 的 Space menu（panel operation）有 `Empty`（清空這個 panel）、`Loading`
（進入 loading，再按一次結束）、`Fail`（這個 panel 出錯；在 `[3] SETTINGS` 上是 `Timeout` 這一欄出錯）、`Disable`（把某些列與動作設成
停用）、`Toast success`、`Toast info`、`Toast warning`、`Toast error`、`Toast long`；I 的 Space menu 有 `Empty`、`Loading`、`Disable`、`Toast info`。
global operation popup 有 `[N]arrow layout`（不管多寬都一次畫一個 panel）、`[1] About`（數字的 global 熱鍵）、`[S]tart task`（畫面
chip 列下的進度條）、`Hide secrets in preview`。icon 佔兩格用 `SHOWCASE__ICON_WIDTH=2` 啟動（E5 的覆寫），statusbar 寫出現在用幾格。
假的檔案在 `/tmp/terminu-showcase`（`project` 與 `.ssh`），file-picker 的打字列打得到；打 `~/` 開頭的路徑看的是真的家目錄。
P `[1] PODS` 上的 `D` 是這個 panel 的 `[D]rag`（M4：panel 的熱鍵優先），要換到 Dialogs 畫面，先換到別的 panel 或用 global operation。

**測試名稱**：`Test` + 規則編號或元件名 + 一句話（例：`TestK4EscOneLayer`、`TestToastReplace`）。L4 的畫面測試是一組跑遍
尺寸、icon 寬度與每一種 popup 的表格測試（`TestL4EveryRowFullWidth`），下面多處引用它。

---

## Principle

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| P3 | 每個動作依作用對象歸 item / panel / global；整批標記算 panel；只跟另一個 panel 有關的只在那個 panel 的 menu；切換畫面是 global | lib：登記動作時一定要給範圍。P `[1] PODS` 的 `[m]ark`（item）與 `Delete marked`（panel）；畫面切換在 global | `TestP3ScopeRequired`、`TestM2SectionsOrder` |
| P4 | 介面元素一個意義：顏色、符號、按鍵、框線、版位不兼職；內容的顏色符號歸 app | lib：介面元素的顏色只從 color 的語意表取（見 color） | `TestColorTokensOneMeaning` |
| P5 | `Space` 固定、`Enter` 概念；panel 熱鍵與領域符號顏色由 app 決定；熱鍵要出現在 menu 用 `[]` 標、不佔 core key、不佔移動字母；元件裡的鍵（表單的 `e`、datetime 的 `t`、textarea 的 `i\|a\|o`）是元件的一部分 | lib：`Enter` 的動作由 app 登記；熱鍵登記檢查（K1、K12） | `TestK1HotkeyCannotTakeCoreKey`、`TestK12HotkeyCannotTakeMovement` |
| P6 | 亮的就是 focus：最上層 popup、focus 的 panel、finder 有 focus 的那塊、同鍵兩處指示會觸發的那個、沒 focus 那塊上的 cursor 淡化 | 見 F8、T2、finder §狀態、M9、color §色碼；預覽框不在其中、一直不淡化（finder §長相） | 各條的測試 |
| 術語 Surface | footer、statusbar、tab 列不是 surface，focus 不停在上面 | 任一畫面 `Tab` 只在 panel 之間 | `TestK2TabCyclesPanels` |
| 術語 輸入態 | 打字列上打的字就是內容；select／picker 的清單、表單本身不是輸入態 | 見 K8 | `TestK8InputState` |
| 術語 模式 | 一部分鍵換意思、`Esc` 離開；zoom 這類版面切換不是模式 | P `[1] PODS` 的 `[D]rag`；D「Note viewer」的 `v`（visual） | `TestK11ModeCoreKeys`、`TestK11ModeLabel`、`TestK11HotkeysSilentInMode` |

## Rules — K Core key

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| K1 | 七種 core key（`Tab` `Enter` `Esc` `Space` `?` `q` `1`–`9`）在每個 surface 意義不變；例外只有輸入態、PTY、模式、表單上的 `q` | 每個畫面、每種 popup | `TestK1CoreKeysEverySurface` |
| K1 | letter hotkey 不能佔用 core key；app 可另定自己的 core key，定了就在所有 surface 一樣 | lib：登記時拒絕（`Run` 也檢查 global 與畫面的鍵）；`App.CoreKeys`：展示場的 `!` 在每個 panel 與 popup 都開 app log，打字列上是字元 | `TestK1HotkeyCannotTakeCoreKey`、`TestK1CustomCoreKeyEverywhere` |
| K1 | 數字鍵：panel 上（沒有 popup 時）跳到目前畫面的第 N 個 panel；沒有那個號碼不作用；換畫面就是那個畫面的 panel | P 按 `1` `2` `3`、按 `4`（不作用）；換到 I 按 `2` | `TestK1DigitsJumpPanels`、`TestK1DigitsPerScreen` |
| K1 | app 不能在 panel 上登記數字熱鍵；popup 裡數字自由（select 選項的數字熱鍵）；數字的 global 熱鍵只在 global operation popup 裡按 | lib：登記時拒絕；I 的 `Profile`（`[1] clock`）；global operation popup 的 `[1] About`（在 panel 上按 `1` 是跳 panel，`?` 也不列它） | `TestK1DigitHotkeyRejectedOnPanel`、`TestK1DigitsFreeInPopups`、`TestM4DigitGlobalOnlyInPopup`、`TestK1DigitGlobalNotInPanelKeyRef` |
| K2 | `Tab`：畫面上換 panel、最後一個回到第一個；不跨畫面、不離開 popup | P 按 `Tab` 走過三個 panel | `TestK2TabCyclesPanels`、`TestK2TabStaysInPopup` |
| K2 | 有動作區的 popup：欄位區與動作區，回來時停在離開前那一項；一定有 `Shift-Tab`；各區裡的項目用 hjkl。表單與 color picker（色版與 `Choose`） | D「Form: New host」、「Form: long」、「Form: with actions」；I 的 `Background`（color） | `TestFormTabBlocks`、`TestFormActionAreaMoves`、`TestColorPickerTab` |
| K2 | 有好幾塊的 popup（finder、datetime picker）：在塊之間 | I `[2] FILES` 的 `[/] Find`；I 的 `Meeting`（datetime） | `TestFinderTabBlocks`、`TestDatetimePicker` |
| K2 | 打字列（finder、select、panel filter）：到清單 | 同上；P `[1] PODS` 按 `/` | `TestFinderTypingTab`、`TestPanelFilterTab` |
| K2 | 只有一塊的 input popup：有灰字提議時接受提議，沒有時不作用 | I 的 `Name`（有舊值）與 `Nickname`（空的） | `TestInputSuggestionTab`、`TestInputNoSuggestionTabNoop` |
| K2 | 多行文字寫入狀態：`Tab` 是縮排；PTY 裡屬於子程序 | I 的 `Comment`（textarea）；T `[1] SHELL` | `TestTextarea`、`TestK10KeysToChild` |
| K3 | `Enter` 做 app 登記的那個動作；同一種項目同一個動作；panel 沒有項目時對整個 panel | P `[1] PODS` `Enter` 開 log；P `[2] LOG` `Enter` 開 viewer（整個 panel 一個動作） | `TestK3EnterRunsRegistered`、`TestK3EnterOnContentPanel` |
| K3 | `Enter` 原則上不移動 focus（元件寫死的：表單送出失敗跳到第一個問題欄位、打字列到清單、明細） | D「Form: New host」；P `[3] SETTINGS` | `TestK3EnterKeepsFocus` |
| K3 | 唯一可以用 `Enter` 換 panel 的是明細：列出項目的 panel 上，`Enter` 可以把 focus 移到顯示那一項明細的 panel；要不要、在哪些項目上由 app 決定。lib：只有 `Enter`、而且 panel 的 `Details` 是同一個畫面的另一個 panel 才收 | K `[1] KINDS` 在 namespaced 的種類上 `Enter` 進 `[2]`；cluster 層級的（`nodes`、`namespaces`…）不進去 | `TestK3EnterToDetails`、`TestK3ToDetailsOnlyEnter` |
| K3 | focus 只在使用者預期會換的時候換：app 的動作（含 `Enter`）可以移動 focus 或換畫面；背景發生的事不可以。lib：`App.Focus`、`App.ShowScreen` 給 app 在動作裡用；增減 panel 時 focus 留在同一個 panel，focus 的 panel 拿掉時給現在在它位置上的 | global 的 `[J]ump to the log` | `TestAppMovesFocus` |
| K3 | 新增一項之後 cursor 移到新的那一項、捲到看得到（lib：`List.Select`，回報那一列看不看得到） | P `[1] PODS` 的 `[n]ew` | `TestListMarkAndSelectByID` |
| K3 | input popup：確定寫回並關；不合格不寫回、錯誤列、不關；值跟打開時一樣、而且打開時就合格才不寫回直接關：打開時是空的、又不能空的，什麼都沒打（或打了又刪光）按 `Enter` 照不合格處理；有建議值時用建議值 | I 的 `Port`（打 `0`）；I 的 `Archive` 按 `Backspace` 再 `Enter` | `TestK3InputConfirm`、`TestK3InputInvalidStays`、`TestK3InputUnchangedNoWrite`、`TestTextareaCheck`（textarea 也一樣：`Textarea.Check`） |
| K3 | 多行文字：寫入狀態 `Enter` 換行，離開寫入後才確定 | I 的 `Comment` | `TestTextarea` |
| K3 | 打字列的 `Enter` 把 focus 移到清單 cursor 那一項、不執行；沒有結果不作用 | I `[/] Find` | `TestFinderTypingEnter` |
| K3 | 其他部分做最直觀的事（月份 → 日曆、R → 選值清單） | I 的 `Meeting`、`Background`（color） | `TestDatetimeEnterMonth`、`TestColorPickerEnterChannel` |
| K3 | 表單的 `Enter` 對 focus 的項目；送出只有按鈕，檢查全部，不合格不送、focus 跳到第一個 | D「Form: New host」 | `TestFormSubmitChecks` |
| K3 | menu、confirm：`Enter` 執行 cursor 那一列 / 接受 | Space menu；D「Confirm」 | `TestMenuEnterRuns`、`TestConfirmEnterAccepts` |
| K4 | `Esc` 關最上層 popup（含 toast）；沒有 popup 時退出模式或往上一層（清 filter）；一次一層；永不離開 app | P：開 filter、進 Drag、疊兩層 popup 後連按 `Esc` | `TestK4EscOneLayer`、`TestK4EscNeverQuits` |
| K4 | 多層 popup 只關最上層，底下原樣 | D「Stack: Space menu → Global operation → Quit」 | `TestK4EscOnlyTop` |
| K5 | panel 上 `Space` 開 / 關 Space menu；`Esc` 也能關 | 任一 panel | `TestK5SpaceToggles` |
| K5 | 其他 popup 上 `Space` 不作用；唯一例外 checkbox 的選項（checkbox popup 的清單、表單上畫出來的 checkbox） | D「Confirm」上按 `Space`；I 的 `Namespaces`；D「Form: open at Port」的 `Agent` | `TestK5SpaceNoopOnPopups`、`TestCheckboxPopupSpace`、`TestFormCheckboxSpace` |
| K5 | popup 的操作揭露在下框 hint 與 `?`，不在 popup 上疊 Space menu | 每種 popup | `TestK5NoMenuOnPopup` |
| K6 | `?` 在任何 surface 都有回應；再按 `?` 或 `Esc` 關；開最前端 surface 的 key reference | 每個畫面、每種 popup、模式裡 | `TestK6HelpEverySurface` |
| K6 | panel：這個 panel 能按的鍵（含移動鍵）＋這裡有作用的 core key ＋ global 熱鍵；popup：只有這個 popup 的熱鍵、`Enter`、`Esc`、移動鍵；開了 `App.ShiftTab` 時 core 段寫 `Tab\|Shift-Tab` | P `[1] PODS` 按 `?`；Space menu 上按 `?` | `TestK6PanelKeys`、`TestK6PopupKeysOnly`、`TestK6ShiftTabInKeyReference` |
| K7 | 別名在所有 surface 一樣有效 | lib：別名登記一次、全域生效 | `TestK7AliasEverywhere` |
| K8 | 輸入態：letter hotkey、`Space`、`?`、`q` 是字元；`Esc` 取消；`Tab` 照 K2；`Ctrl-C` 進離開流程；離開輸入面立刻恢復 | I 的 `Name` 打 `q ?`；P filter 打 `q` | `TestK8InputState`、`TestK8CtrlCInInput` |
| K8 | textarea 寫入狀態 `Tab` 插入 `input_tab_width` 個空白（E3 的設定，預設 4），不插 `\t` | I 的 `Comment` | `TestTextarea` |
| K9 | `q` 與 `Ctrl-C` 同一個離開流程；`q` 在輸入態是字元、在表單上不作用；`Ctrl-C` 在輸入態與表單上有效；PTY 裡兩個都屬於子程序 | 任一 panel；D「Form: New host」上按 `q` | `TestK9QuitSameFlow`、`TestK9QNoopOnForm` |
| K9 | 離開流程由 app 決定（直接 / 先確認 / 選擇）；進行中再按 `Ctrl-C` 立刻離開；離開列在 global operation popup。lib：`App.Quit`（confirm）或 `App.QuitFlow`（app 自己的動作，例：開一個 menu 選離開的方式） | 展示場：先 confirm（`Quit showcase?`）；D「Quit: the app’s own flow」之後按 `q` 是 menu | `TestK9SecondCtrlCQuits`、`TestK9QuitInGlobalPopup`、`TestQuitFlowOfTheApp` |
| K9 | app 自己的 popup 有東西沒存時（`QuitNote`），離開的 confirm 最後加一句 Peach，`App.Quit` 是 nil 也先問；離開的 confirm 被底下的 popup 帶著關掉時，離開流程結束、下一次 `Ctrl-C` 再問 | D「Popup: the app’s own」畫一格再 `Ctrl-C`（`Changes to icon are not saved.`） | `TestAppPopupQuitNote`、`TestK9QuitConfirmClosedUnder` |
| K10 | PTY 裡按鍵都給子程序；core key 與熱鍵失效 | T `[1] SHELL` 裡按 `Space ? q Tab Esc 1` | `TestK10KeysToChild` |
| K10 | 至少一個出口鍵（`Alt-Esc`），focus 在 PTY 時常駐揭露 | T：footer `Alt-Esc:leave …`；terminal popup 下框 | `TestK10ExitKeyShown` |
| K10 | `Alt-Esc` 離開或結束時一律先 confirm（`Enter` 離開、`Esc` 回 PTY）；PTY 裡面的動作不用問 | T `[1] SHELL`、terminal popup | `TestK10AltEscConfirms` |
| K10 | 其他 Alt 出口鍵要不要 confirm、保留哪些鍵、出口後 focus 落點：app 決定（lib 提供選項） | T `[2] SESSIONS` 的 `[Alt-t]erm` 隱藏 terminal popup | `TestK10ReservedKeysShown`、`TestTerminalKeyTakesSeveral`（`TerminalKey.Keys`：一組鍵寫一次） |
| K10 | 子程序還沒準備好時可以不轉送一般鍵，但 `Ctrl-C` 照轉、出口鍵照有效照揭露 | T `[1] SHELL` 的 `Slow start`（模擬連線中） | `TestK10NotReadyForwardsCtrlC`、`TestTerminalFirstOutput`（lib：`Terminal.OnFirstOutput`、`Spoke`，app 拿來結束 NotReady 與 loading） |
| K10 | `Alt-Esc` 與兩次 `Esc` 的 byte 一樣（實作參考：bubbletea v1.3.10 實測會黏）：C2 驗 | 見紀錄 C2 | — |
| K11 | 模式裡 `Space` 不作用；`?` 開模式的 key reference；`Esc` 離開、回到進入前的狀態；`q`、`Ctrl-C` 照 K9 | P `[1] PODS` 的 `[D]rag` | `TestK11ModeCoreKeys` |
| K11 | 模式裡只有模式自己的鍵與上表的 core key 作用：panel 熱鍵與 global 熱鍵都不作用，按了不出聲 | Drag 裡按 `m`（panel 熱鍵）、`I`（global 熱鍵） | `TestK11HotkeysSilentInMode` |
| K11 | 模式可以暫停 `Tab` 與數字鍵，按了要有回應（toast）；toast 在時第一個 `Esc` 先收 toast | Drag 裡按 `Tab`、`2` | `TestK11TabPausedToast`、`TestK11DigitsPausedToast` |
| K11 | 模式名在框的上框右側、外框換模式色、離開恢復；不只靠顏色 | Drag；D「Note viewer」的 `v`；I 的 `Segments` 按 `D` | `TestK11ModeLabel`、`TestReorderDrag` |
| K11 | popup 裡的模式（reorder 的 Drag、viewer 的選取模式）也一樣：只有模式的鍵與上表的 core key 作用，app 自己的 core key 也不作用 | I 的 `Segments` 按 `D` 再按 `!`；D「Note viewer」按 `v` 再按 `!` | `TestReorderDragModeKeys`、`TestK11PopupModeCoreKeys` |
| K12 | 不在打字的地方 `j k u d g G h l` 保留給移動，任何動作都不佔 | lib：登記時拒絕（panel 與 menu；popup 裡數字可以） | `TestK12HotkeyCannotTakeMovement`、`TestK12HotkeyCannotTakeMovementInMenu`、`TestK1DigitsFreeInPopups` |
| K12 | 有左右結構：`h\|l` 左右（切 tab、跨側、日曆前後一天） | P `[1] PODS` 的 tab；I 的 `Meeting` | `TestK12HLSwitchTabs`、`TestDatetimePicker` |
| K12 | 一維清單：`h\|k` 往前、`j\|l` 往後；在有 tab 的 panel 裡 `h\|l` 給 tab；reorder 也是（含 Drag）；key reference 照這樣寫（表單、沒有 tab 的一列一個值、menu、reorder 寫 `h\|k`、`j\|l`） | D「Form: New host」；P `[3] SETTINGS`；I `[1] INPUTS`；I 的 `Segments`；各處按 `?` | `TestK12OneDimensional`、`TestReorderOneWay`、`TestKeyReferenceMoveSection`、`TestK6PopupKeysOnly` |
| K12 | `u\|d` 半頁、`gg\|G` 頭尾；單按 `g` 不作用，後面接沒登記成組合的鍵時當 `g` 沒按過（`g` 然後 `j` 往下一列）；其他 `g` 開頭可當熱鍵；不收 `Ctrl-U\|Ctrl-D` | P `[1] PODS`（40 列）；展示場登記一個 `[go]to` | `TestK12HalfPageAndEnds`、`TestK12SingleGNoop`、`TestK12NoCtrlUD` |
| K12 | `Home`、`End` 只在打字列（游標到最前、最後）；清單、menu、viewer、textarea 的「移」都不作用 | 任一清單、menu、D「Note viewer」、I 的 `Comment` 上按 `Home`、`End` | `TestK12HomeEndOnlyTyping` |
| K12 | 一步一步的移動（`j\|k`、`h\|l`、方向鍵、`Tab`）到頭尾繞回：panel 清單、一列一個值、menu、表單、finder／select／checkbox popup、panel 的 tab、picker 的欄與項目 | P `[1] PODS` 在最後一列按 `j`；tab 在最後一個按 `l`；每種清單各一次 | `TestK12StepWraps` |
| K12 | Drag mode 拿著的一項也繞回（panel 的 Drag、reorder）：拿著最後一項往後到第一個、其他各往後一格；拿著第一項往前到最後一個 | P `[1] PODS` 按 `D` 再按 `k`；I 的 `Segments` 在最後一項按 `D` `j` | `TestPanelDragHeldRow`、`TestReorderDragWraps` |
| K12 | `u\|d` 到頭尾停住、不繞；照空間走的不繞：日曆的日子、月與年、左右兩側、一段文字裡的移動 | P `[1] PODS` 在倒數幾列按 `d`；I 的 `Meeting` 日曆；D「Note viewer」的 visual | `TestK12HalfPageStops`、`TestK12SpatialNoWrap` |
| K12 | 一段文字裡（viewer 的選取模式、textarea 的移動狀態）加 `w\|b\|e`、`0\|$`；viewer 平常 `h\|l` 給 tab，選取模式裡 `h\|l` 移游標、不切 tab | D「Note viewer」的 visual；D「Popup with tabs」（平常 `h\|l` 切 tab，`v` 之後 `h\|l` 移游標）；I 的 `Comment` 移動狀態 | `TestK12TextMotions`、`TestNoteHotkeysAndMode` |

## Rules — M Menu 與揭露

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| M1 | 非輸入態的每個畫面常駐顯示 `?` 與 `Space`（模式裡不必顯示 `Space`） | footer，每個畫面（popup 開著時照底下畫面的寫） | `TestM1FooterEntries` |
| M2 | Space menu 三區：`item operation` → `panel operation` → 一列 `Global operation`（沒熱鍵），順序固定、標題英文原字（畫成大寫，components §大寫） | 任一 panel 按 `Space` | `TestM2SectionsOrder`、`TestM2HeadingStrings` |
| M2 | global 那一列不加標題、跟上面用分隔線隔開；沒有對象就沒有那一區；item 與 panel 一律加標題（只剩一區也加） | P `[1] PODS` 的 `Empty` 之後按 `Space`（沒有 item 區） | `TestM2NoTargetNoSection`、`TestM2HeadingAlways` |
| M3 | 每個 item / panel operation 都在 Space menu；每個 global operation 都在 global operation popup；popup 的操作在 hint 與 `?` | lib：只有登記過的動作才有熱鍵 | `TestM3EveryHotkeyInMenu`、`TestValuesEnterInMenuAndKeys`（一列一個值的 `Enter` 也在 Space menu 與 `?`） |
| M3 | 只有打字列的 popup（text、password、number）：hint 列出全部操作、80 欄放得下 | I 的 `Name`、`Password`、`PIN`、`Port` | `TestM3TypingPopupHintFits80` |
| M4 | global operation popup：從 Space menu 的 global 列 `Enter`，疊在 Space menu 上；一種 menu；全部全域動作；離開在這裡 | 任一 panel：`Space` → `Global operation` | `TestM4GlobalPopup` |
| M4 | `Esc` 回到 Space menu；執行了照 T1；目前畫面的切換列停用 | 在 `[P]ANELS` 上看 `[P]anels` 那一列 | `TestM4EscBackToSpaceMenu`、`TestM4CurrentScreenDisabled` |
| M4 | global 熱鍵在 panel 上直接按（沒有 popup 時）；跟 panel 熱鍵撞鍵時 panel 的優先、那個 panel 的 key reference 不列被蓋掉的 global 鍵；數字的 global 熱鍵只在 popup 裡；模式裡不作用 | P `[1] PODS` 的 `[D]rag` 蓋掉 global 的 `[D]ialogs` | `TestM4GlobalHotkeyOnPanel`、`TestM4PanelKeyWins` |
| M4 | key reference 唯讀、可捲、沒游標、不能執行 | 任一處按 `?` | `TestKeyReferenceReadOnly` |
| M5 | 每一列：左邊動作名稱、右邊一句單行說明 | Space menu | `TestMenuRowLayout` |
| M5 | 熱鍵標記：字母出現在 label 裡（不分大小寫）原地包第一個，括號裡照鍵寫、label 照原本的大小寫：`[r]ename`、`[n]ew`、`D[e]lete`、`UR[L]`、`Ex[P]ort`；不在 label 裡放前面 `[x] Reset`；modifier 照同一套 `[Alt-t]erm`；多字元 `[go]to`；數字放前面 `[3] Favorites`；core key 放前面 `[Enter] Edit`；沒熱鍵不加 | 展示場的登記各用到一次（見下面「熱鍵標記」） | `TestM5HotkeyMarking` |
| M5 | 括號裡印的就是要按的鍵（大小寫算數）；label 已寫出鍵的不再括；不只靠顏色或 glyph 暗示 | 同上 | `TestM5HotkeyMarking` |
| M5 | 鍵名：鍵帽名、大駝峰（`Esc` `Tab` `Enter` `Space` `Backspace` `Delete` `Home` `End` `PgUp` `PgDn`、`↑↓←→`）；`Ctrl` 後字母大寫；`-` 連 modifier；幾個鍵做同一件事用 `\|` 並列（不用 `/`）、`–` 範圍 | lib：鍵名由一個函式產生 | `TestM5KeyNames` |
| M5 | 依位置：label 用括號；句子裡鍵加方括號；hint、footer `鍵:說明`、項目間一格；key reference 兩欄、不加括號冒號 | 空狀態、toast、hint、footer、`?` | `TestM5WritingByPosition` |
| M5 | hint、footer 的鍵一色（Blue）、冒號與說明一色（Overlay0）；並列的 `\|` 用說明的顏色（Overlay0），key reference 的鍵欄也是 | 每個 hint、footer、`?` | `TestM5HintColors`、`TestM5KeysJoinedWithBar` |
| M6 | 對象不存在：那一列（區）不出現；存在但現在不能：停用色、說明不變、cursor 跳過、熱鍵不作用 | P `[1] PODS` 的 `Disable` | `TestM6DisabledRow` |
| M6 | key reference 照同一套（停用的鍵照列、停用色）；hint 與 footer 只列現在按得了的也可以（app 決定），例外是元件規定一定要拿掉的：表單的按鈕還在跑時、finder 沒有結果時、panel 的清單一項都沒有時 | `Disable` 之後按 `?`；hint：P `[1] PODS` 的 `m:mark` 變停用色，I `[2] FILES`（`HintPressableOnly`）的 `V:view` 不列；三個例外見 form §按鈕、finder §hint、panel §框 | `TestM6KeyReferenceDisabled`、`TestM6HintListsDisabled`、`TestFormActionWhileRunning`、`TestFinderHintNoResults`、`TestPanelHintNoItems`、`TestFormKeyReferenceFollowsFields`（表單的 `Space` 只在有 checkbox 選項時列、`e` 只在有欄位有問題時列）、`TestM6HiddenGlobalNotInKeyReference`（global 動作的對象不存在時，app-wide 段不列） |
| M6 | 清單空了沒有東西可以移、內容整個放得下沒有東西可以捲，都算對象不存在：移動鍵、捲動鍵不列（List、Pager、Values、confirm、error popup 與其他 note） | P `[2] LOG` 只有幾行時按 `?`；I `[2] FILES` 的 `Empty` 後按 `?`；D「Confirm」上按 `?` | `TestKeyReferenceMoveSection`、`TestKeyReferenceScrollOnlyWhenItScrolls` |
| M6 | key reference 裡講別的 surface 的段落照常顯示，排在最後（`app-wide` 之後） | T `[1] SHELL` 的 `?` 最後一段 `TERMINAL` | `TestM6OtherSurfaceSection` |
| M7 | focus 在 panel 上時 `Space`、`?` 一定有反應；popup 上一定有反應的是 `?` | P `[2] LOG`（沒有 item）按 `Space` | `TestM7PanelEntriesRespond` |
| M8 | 其他 menu 要分組時第一組是 cursor 相關；每一組都加標題，第一組也是（lib：`MenuSection.Title`） | D「Menu: Open with」 | `TestM8GroupedMenuOrder` |
| M9 | 同鍵兩處指示同時看得到時，會觸發的亮、不會的暗 | P：`[1] PODS`、`[2] LOG`、`[3] SETTINGS` 的下框 hint 都有 `Enter` | `TestM9ActiveHintBright` |

## Rules — L 版面

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| L1 | 80 × 40 能完成所有核心任務；放不下所有 panel 時照 screen 的窄寬 | 終端機開 80 × 40 走一遍；`Narrow layout` | `TestL1Core80x40`、`TestScreenNarrow`、`TestOnResize`（lib：`App.OnResize`，app 照寬度重排） |
| L1 | 最低尺寸預設 80×40、app 可以調低不能高於（lib：`App.MinWidth`、`MinHeight`）；比它小時只在正中央畫名稱（Blue 粗體）與 `needs a terminal of at least W×H`（Overlay0），放不下先拿掉提示、名稱畫得下多少畫多少，到 1×1 不當掉；按鍵都不作用、不送進 PTY，只有連按兩次 `Ctrl-C` 立刻離開；狀態保留，拉大回到原本的畫面 | 展示場的最低尺寸是 60×20：把視窗縮到比它小 | `TestL1LeastSize` |
| L2 | 標題、statusbar、chip 等動態文字固定寬度；一列裡最後一個動態欄位可以不補空白 | statusbar 的錯誤與警告數（`%-3d`，固定寬）、最後的 `copying: 4%`（不補）；panel 標題換 tab | `TestL2FixedWidthChrome` |
| L3 | footer、statusbar、tab 列列數鎖死；放不下截斷或從尾端捨，不折行 | 縮到 30 欄 | `TestL3ChromeRowsFixed` |
| L4 | 每一列剛好等於終端機寬度 | 每個畫面、每種 popup，多種尺寸（含 5 列以下與 1×1：popup 被截掉、不 panic），icon 一格與兩格；app 的文字裡有 Tab 時一樣 | `TestL4EveryRowFullWidth`、`TestE3TabsDrawnAsSpaces` |
| L5 | focus 不只靠顏色（粗線與細線）；切換 focus 內容不位移 | `Tab` 換 panel；Drag 模式裡照樣看得出 focus | `TestL5FocusNotColorOnly`、`TestL5NoShift` |

## Rules — F Popup

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| F1 | 七類，每類按鍵意義固定（menu、confirm、input、form、note、toast、terminal） | menu、confirm、form、note、toast：D 各有幾列；input：I `[1] INPUTS`；terminal：T `[2] SESSIONS` | 各元件的測試 |
| F1 | note 可以有自己的熱鍵與模式；有可用 `Enter` 執行的清單就是 menu | D「Note viewer」（`/` `y` `v`） | `TestNoteHotkeysAndMode` |
| F1 | menu 的 `Enter` 可以開那一列的全文（note），仍是 menu | D「Menu: Worklist」 | `TestMenuEnterOpensFullText` |
| F1 | confirm 可以帶回答前要看的內容，長時 `j\|k` 捲 | D「Confirm: with details」「Confirm: long」 | `TestConfirmScrollsContent` |
| F1 | 依階段換類別、同一時間一類；`Tab` 切、`Esc` 關整個；預覽不拿 focus；只有有 focus 的那邊亮 | I `[/] Find` | `TestFinderPhases` |
| F1 | 清單加搜尋就是 finder（select、checkbox popup、file-picker） | I 的 `Profile`、`Namespaces`、`Identity file` | 見 finder |
| F1 | 沒有搜尋的清單是 menu，只有 reorder 例外：它搬動項目、確定後寫回整個順序，是 input | I 的 `Segments` | 見 reorder |
| F1 | toast 不是一層（不淡化、不算層數），但 `Esc` 先收它 | D「Toast over a popup」 | `TestToastNotALayer` |
| F1 | 多步驟流程每一步是自己的 popup | D「Menu: Sort by」（欄位 → 方向） | `TestMultiStepStacks` |
| F1 | 七類以外的 popup 是 app 自己的一類，app 記成偏離（例：kamonu 畫格子的 popup）。lib 給入口 `App.Popup`：框裡的內容與按鍵歸 app；框、標題（`DRAW: icon`）、hint、層色、開關動畫、淡化、F7 的大小歸 lib；app 沒收的 `Esc` 關這一層、`?` 開它的 key reference（只列它自己的鍵）、`q` 進離開流程，`Ctrl-C` 不經過它；lib 的 confirm、input、error popup 疊在它上面，`ClosePopup` 連同上面的一起關 | D「Popup: the app’s own」：`hjkl` 移、`Enter` 畫、`Tab` 到 Save、改過再 `Esc` 先問、`q` 不作用 | `TestAppPopupFrame`、`TestAppPopupSize`、`TestAppPopupKeys`、`TestAppPopupStack`、`TestAppPopupComparable` |
| F2 | 開關都有動畫，全家族同一種：中間拉一條線、再上下展開，收起倒著播；每格 32ms、各 9 格；打開的動畫播完前不收鍵 | 每種 popup | `TestF2AnimationFrames`、`TestF2NoKeysWhileOpening` |
| F2 | 好幾層一起收起時（一個動作關掉整疊、T1 清掉兩層以上的 source）每一層同時收、各自照自己的大小與層色，總共還是 9 格 | D：Space menu → Global → `?` 再跑會關掉整疊的動作 | `TestF2LayersCloseTogether` |
| F3 | `Esc` 立刻開始關閉任何 popup（含 toast）；關閉照常有動畫；在關的 popup 不再理會 `Esc` 與其他鍵 | 連按兩下 `Esc` 關兩層 | `TestF3EscImmediate`、`TestF3ClosingIgnoresKeys` |
| F3 | 例外：PTY（toast 只能等時間到）、textarea 寫入狀態、改過的表單、textarea 與 reorder 先 confirm | T；I 的 `Comment`；D「Form: New host」改過再 `Esc`；I 的 `Segments` 搬過再 `Esc` | `TestF3Exceptions`、`TestReorderEsc` |
| F4 | 從 A 開 B，A 留著；取消 B 回 A；多層一樣 | D 的疊法情境 | `TestF4KeepSource`、`TestInputSteps`（多步驟的 input：`Stay`，`Esc` 退一步、值都在） |
| F5 | 錯誤立刻出現（toast 或 popup）、`Esc` 可關、不阻塞 app；有沒有錯誤歷史由 app 決定 | P `[1] PODS` 的 `Fail`；D「Error popup」 | `TestF5ErrorNonBlocking` |
| F6 | 要不要 confirm 由 app 決定，決定了每次都問；經 picker 明確選的可算已確認 | lib：動作登記「先問」 | `TestF6ConfirmEveryTime`、`TestF6AskFor`（`Action.AskFor`：問句執行時才產生、回 nil 不問；從 menu 跑時 `Esc` 回 menu）、`TestOnCancel`（`OnCancel`：沒回答就關掉時告訴 app） |
| F6 | `Enter` 接受、`Esc` 取消，回答只有這兩個鍵（沒有 `y`、`n`）；寫出接受會做的事；不能被誤觸完成 | D「Confirm」 | `TestConfirmKeys`（`y`、`n` 不作用，`?` 只列 `Enter`、`Esc`） |
| F7 | 寬度兩種：內容確定 = 最寬列 + 4、至少放得下標題與 hint、最多 `min(寬 − 2, 120)`；不確定 = `min(寬 − 2, 120)`；水平置中；開著不變；有整列 cursor 的，右邊的留白跟箭頭一樣寬（powerline 符號寬）；寬度至少 hint + 6（PIN 43 格） | 每種 popup；寬 200 的終端機看 120 上限 | `TestF7WidthKnown`、`TestF7WidthKnownUnlockContent`、`TestF7WidthUnknown`、`TestF7WidthCap120` |
| F7 | 高度打開時定、最高畫面高 − 2（chip 列或 statusbar 與 footer 看得到，chip 列下面的分隔線可以被蓋住），超過捲；打開時不知道會多高的（有預覽的 finder、textarea）一律取最高；loading 期間與使用者操作造成的改變可以變；畫面太矮時 popup 也不超過自己的高度（框裡的空白列先讓，terminal 例外） | D「Menu: long」；I `[/] Find` 打字篩選；把視窗縮到 10 列開表單 | `TestF7HeightFixed`、`TestF7HeightMax`、`TestF7ShortScreenKeepsFooter` |
| F7 | loading 時標題後一定有輪轉 icon；只看整個 popup 的內容 | D「Loading popup」 | `TestF7LoadingIcon`、`TestF7LoadingGrows`、`TestLoadSelectAndChecks`（lib：`LoadSelect`、`LoadChecks`） |
| F7 | 垂直置中；toast 的下框在畫面倒數第三列 | 每種 popup；D「Toast」 | `TestF7Position`、`TestToastPosition` |
| F7 | 值可能不合格的 popup 預留一列錯誤列；動作失敗開 error popup | I 的 `Port`；D「Form: save fails」 | `TestF7ErrorRowReserved` |
| F7 | terminal：寬 `寬 − 2`；上面留 statusbar 或 chip 列（含分隔線）、下面到最後一列，footer 不畫；上下左右都不留白 | T `[2] SESSIONS` 的 `[Alt-t]erm` | `TestTerminalPopupSize` |
| F8 | 最上層以外全部淡化（底下 popup 與 base 畫面），一直只有最上層亮 | D 的三層疊法 | `TestF8DimAllButTop` |
| F8 | 在收的 popup 不算一層（color 的層色）：層色只數底下開著的層，表單送出後開的下一個 popup 從第一格就是第 1 層色；壓在最上層底下、還在收的也淡化。表單送出只關表單自己，送出時開的 popup 留著 | 用 lib 的 app：kamonu 的 `NEW KAMON` 按 Create 開 `DRAW`（展示場沒有這種流程） | `TestAppPopupFromFormSubmit` |
| F8 | 串流內容與警示色一起淡化；淡化照 color 的算法改每一個顏色、形狀不動 | P `[2] LOG` 上開 popup | `TestF8DimStreamToo`、`TestDimEveryColor` |
| F8 | 對畫好的畫面淡化一次（失焦 panel 更暗）；toast 不觸發；底下 popup 的框是自己層色的淡化 | 同上 | `TestF8DimOnce`、`TestF8BorderKeepsLayerColor` |

## Rules — X、T、S

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| X1 | 沒有 mouse 也能完整操作 | 全部用鍵盤走過 | 整份清單的按鍵測試 |
| X2 | 有 mouse 時每個動作對應一個鍵盤動作 | lib 先不做 mouse；做的時候補 | （做 mouse 時補） |
| T1 | 預設保留 source；照「確定之後的動作」判：打開來看、選值寫回、用編輯器改保留；讓那一項不在、開 shell、換畫面、drill-down 清掉。lib：從 menu 跑的動作，打開了 popup 就把 menu 留在底下（`Action.Clear` 讓它清掉 menu 與底下的 popup），什麼都沒打開就關掉 menu；`Action.Stay` 一律留著 | D 的疊法；P `[1] PODS` 的 Space menu 跑 `[V]iew` 類的動作；`[2] SESSIONS` 的 `[Alt-t]erm` 從 Space menu 打開會清掉 menu | `TestT1KeepOrClear`、`TestT1KeepByDefault`、`TestCloseAll`、`TestInputSteps`（lib：`App.CloseAll`、`App.CloseTop`） |
| T1 | confirm 只是門：在 confirm 上 `Esc` 一律回到 source；接受之後照它後面的動作判；menu 的動作失敗開的 error popup 關掉回到 menu | P `[1] PODS` 的 Space menu 跑 `[x] Delete` 再 `Esc` | `TestT1KeepByDefault` |
| T1 | 清掉 source 時，target 打開的同時 source 收起，兩段動畫一起播、不等 source 收完；在收的 source 不算一層，target 從第一格就是自己那一層的層色，source 收完時不跳色（layout/popup 的開關動畫、color 的層色） | `[2] SESSIONS` 的 `[Alt-t]erm` 從 Space menu 打開，看 terminal 的框一路都是第 1 層色 | `TestT1SourceClosesWhileTargetOpens`、`TestAppPopupFromFormSubmit` |
| T2 | 失焦 panel 用 F8 的淡化變暗；串流內容失焦不變暗，但串流 panel（含 PTY）下框的 hint 照樣變暗；popup 蓋上時一律淡化 | P：focus 在 `[1]`，`[2] LOG` 內容不暗、hint 暗，`[3]` 暗 | `TestT2UnfocusedDim`、`TestT2StreamNotDimmed` |
| S 開頭 | splash 佔滿整個畫面，原本的畫面都不畫 | P `[1] PODS` 按 `V` | `TestS1SplashOnV`、`TestS5SplashFillsScreen` |
| S1 | 在 panel 上按 `V` 開 splash；某 panel 可把 `V` 給自己用，至少一個 panel 仍是 splash | P `[1] PODS` 按 `V`；I `[2] FILES` 的 `V` 給 `[V]iew` | `TestS1SplashOnV`、`TestS1EveryAppHasSplash` |
| S2 | 不出現在 Space menu、global operation popup、key reference、footer、hint、README | lib：splash 沒有登記項 | `TestS2SplashNotDisclosed` |
| S3 | 任意鍵只關掉 splash（含 `q`、`Ctrl-C`、`Esc`、`Space`、`?`） | splash 上按 `Ctrl-C` | `TestS3AnyKeyClosesOnly` |
| S4 | 啟動不播；popup 開著、輸入態、PTY 裡按 `V` 不叫出 | 開著 popup、在 filter 打字、在 Shell 裡按 `V` | `TestS4SplashOnlyOnPanel` |
| S5 | 畫 icon 一格對一格：25 × 19 格加上下各一列底共 21 列，一格一個 `nf-fa-square` 佔兩欄（icon 佔一格時補空白），底 Surface0、U 框 `#205090`、字 `#f2b753`；下面空一列接六列字（名稱 Blue 粗體、版本 Blue、介紹 Text、`developed by` Overlay0、信箱金色、`Press any key to close` Overlay0），在 mark 的寬度裡置中、每列不超過 50 欄（超過時建立 splash 回錯誤）；整塊上下左右置中；E8：展示場畫 tdp 的 `principle/icon.svg`（複本在 `examples/showcase/icon.svg`） | P `[1] PODS` 按 `V` | `TestS5SplashMatchesIcon`、`TestS5SplashRefusesBadIcon`、`TestS5SplashRefusesWideText` |
| S5 | 放不下整塊（不到 50 欄或 30 列）時不畫 mark、只畫字，置中、照樣分兩批；太窄照 components 的「折行」；一個字比終端機寬就照格子切開、每段置中；還是太矮從下面往上截（提示 → 作者兩列 → 介紹，連同上面的空白列），名稱與版本一定出現，只有 1 列時只畫名稱 | 把視窗縮到 49 欄或 29 列以下按 `V` | `TestS5SplashTooSmall`、`TestS5SplashOneRow` |
| S5 | 揭露動畫：tick 10ms、段間停 250ms；底一個 tick 一列由上往下 → 金色的字照 svg 裡註解的順序一個字一段（一個註解開始一個字，第一個是 `<!-- U -->`）、一個 tick 2 格打散 → U 框由下往上一列一列、同一列從中間往兩端，一個 tick 一對（中線那格自己一個 tick） → mark 畫完 400ms 後名稱、版本、介紹，再 500ms 後作者兩列與提示；照打開後經過的時間走，晚到的 tick 不累積延遲 | 同上 | `TestS5SplashReveal`、`TestS5SplashKeepsTime` |
| S1 | lib：app 能隨時播一個指定的 splash（`App.PlaySplash`，例：kamonu 看設計中的 mark）；照樣任意鍵關掉；展示場不揭露（S2） | — | `TestPlaySplash` |

## Rules — E App 與環境（分工照 E8）

| 出處 | 規定 | 怎麼驗 | 測試 |
|---|---|---|---|
| E1 | 展示場：`showcase version`（`--version`、`-v`）印版號；`showcase help`（`-h`、`--help`）印命令列用法；lib 不適用 | 執行四種寫法 | `TestE1VersionHelp` |
| E2 | 變數名 `<APP>__<NAME>`；lib 讀的變數掛在 app 的名字底下；`TERMINU__ICON_WIDTH`、`TERMINU__POWERLINE_WIDTH` 給子程序 | lib 提供變數名的函式；展示場只讀 `SHOWCASE__…` | `TestE2EnvNames` |
| E3 | 設定與資料的位置（v0.6.4 補了 Windows）：lib 不適用，展示場不存東西；file-picker 的 `~/` 是使用者的家目錄（`os.UserHomeDir`，Windows 上就是 `%USERPROFILE%`）；家族共用的設定 key `hide_secrets_in_preview`（預設 `false`）、`input_tab_width`（預設 `4`）由 app 讀，交給 lib 的 `App.HideSecretsInPreview`、`App.InputTabWidth`；畫在畫面上的內容裡的 Tab（viewer、檔案預覽、清單、menu……）畫成 `input_tab_width` 格空白，表單與 panel 上的值、輸入框的打字列例外：畫成紅色的 `\t` | 展示場的 global `Hide secrets in preview`；I 的 `Separator` | `TestFilePreviewHidesSecrets`、`TestTextarea`、`TestE3TabsDrawnAsSpaces`、`TestValueControlChars` |
| E4 | 需要 Nerd Font 與 truecolor（lib 的 README 寫明）；PUA glyph 在程式碼裡寫成 code point | 看 README；掃原始碼找不到 PUA 字元 | `TestE4NoLiteralPUA` |
| E5 | 量的是游標實際前進幾格；所有量寬度的地方走同一個函式（含疊 popup）；畫到終端機用 bubbletea v1 的 renderer（紀錄 C1） | `SHOWCASE__ICON_WIDTH=2`；`proto/c1/tmux-check.sh` 的作法 | `TestL4EveryRowFullWidth`（icon 兩格）、`TestE5OutputDropsEraseRight`（renderer 補的「清到行尾」拿掉，紀錄 C1）、`TestVTWideCharacters`（PTY 裡的字也照同一個寬度放） |
| E5 | icon 與 powerline 符號（U+E0A0–U+E0D7）各量各的；啟動時 CPR 探測（unix）；順序 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → `LC_TERMINU_ICON_WIDTH`（隔著 ssh 帶過來的）→ 探測，powerline 照 `<APP>__POWERLINE_WIDTH` → `TERMINU__POWERLINE_WIDTH` → `LC_TERMINU_POWERLINE_WIDTH` → 探測；Windows 預設一格 | 在 icon 佔兩格的 tmux 裡啟動看 statusbar（寫出兩種寬度） | `TestE5WidthOrder`、`TestE5ProbeCPR` |
| E5 | 有 PTY 的 app 開子程序時設 `TERMINU__ICON_WIDTH`、`TERMINU__POWERLINE_WIDTH`（自己的格數） | T `[1] SHELL` 裡 `echo $TERMINU__ICON_WIDTH $TERMINU__POWERLINE_WIDTH` | `TestE5ChildEnv` |
| E5 | 疊 popup 不當掉：比畫面寬、高、兩邊都大都切，不 panic | 縮小終端機再開 popup | `TestE5CompositeOversized` |
| E6 | lib：git tag，沒有 binary；展示場不發布；L4 的畫面測試兩邊都要；E6 指向的 project（v0.7.0：app 的 repo 怎麼排、怎麼發布）lib 與展示場不套，照 E8 的表 | — | `TestL4EveryRowFullWidth` |
| E7 | lib 的 README 自己的結構（安裝 `go get`、快速開始是程式碼、寫明展示場怎麼跑；設定段寫出 E3 的共用設定 key 怎麼交給 lib）；dev-remarks 同一個骨架 | 對照這個 repo 的 README 與 `docs/dev-remarks.md` | — |
| E8 | 分工表；lib 不偏離 tdp | dev-remarks「偏離 tdp」是空的；做不到的都在紀錄裡 | —、`TestCanvas`（lib：`App.Canvas`，lib 管不到的畫面照樣用 confirm、toast、input） |

## color

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| color §呈現規則 1、3 | 三個錨點推導其餘；每個意義一個顏色 | lib：顏色只從語意表取 | `TestColorTokensOneMeaning` |
| color §呈現規則 2、§popup 的層色 | 第 K 層 = `lerp(Lavender, Sapphire, K/4)`，4 層以上固定；toast 一律第 1 層 | D 的四層疊法（Space menu → Global → Quit → `?`） | `TestLayerColors`（每個通道四捨五入，等於表 `#A4C0FA` `#94C3F5` `#84C5F1` `#74c7ec`） |
| color §呈現規則 4 | 警示色不跟層數換色，蓋在底下照 F8 淡化 | D「Confirm: with details」的 Peach 在第 1 層；P `[1] PODS` 的 Space menu → `[x] Delete` 在第 2 層 | `TestWarningColorFixed` |
| color §呈現規則 6 | 粗體只用在：cursor 上的字、popup 標題、膠囊的字、模式名、標題後的 loading icon | 每種畫面 | `TestBoldOnlyWhereAllowed` |
| color §色碼 | 每個意義的色碼與用處（Base、Blue、Surface2、層色、Lavender、Green、Yellow、Peach、Red、Overlay0、Text、Subtext1、Surface1） | 每個畫面 | `TestColorTable` |
| color §色碼 | 畫面不塗底色：沒指定背景的格子留給終端機；Base 只用在色塊上的字、膠囊接縫沒填色的那一邊、淡化的目標 | 換一個不是 Base 的終端機底色看 | `TestNoBackgroundPainted` |
| color §色碼 | 模式色 Yellow 也用在 Drag 拿著的那一列 | P `[1] PODS` 的 `[D]rag`；I 的 `Segments` 按 `D` | `TestPanelDragHeldRow`、`TestReorderDrag` |
| color §色碼 | 例外是色塊：內容本身就是顏色的格子（`ColorText` 的 `███`）在 cursor 色塊上保留自己的顏色，同一塊上的其他字照樣換成 Base。lib：表單、一列一個值的 panel、一般清單的 cursor | I `[1] INPUTS` 的 `Background` 停在上面 | `TestSwatchOnCursor` |
| color §色碼 | cursor 色塊：popup 層色底、panel Subtext1 底，字 Base 粗體；沒 focus 那塊的 cursor 再淡化一次；放在色塊上的字一律 Base | P 的 cursor；I `[/] Find` 的兩塊 | `TestCursorBlock` |
| color §淡化 | `dim(c) = c × 0.45 + base × 0.55`，每個通道四捨五入，前景背景都算；256 色的 16–255 照標準值；0–15 用終端機的調色盤（啟動時 OSC 4 問，問不到用 xterm 的值） | F8、T2 的畫面；在 tmux 裡改調色盤看 | `TestDimFormula`、`TestDimPaletteFromOSC4`、`TestDimPaletteFallbackXterm` |
| color §淡化 | 絕不變亮（每通道取小）；一律輸出 24-bit；沒指定前景給 `dim(Text)`；bold、reverse、游標移動、文字不動 | 同上 | `TestDimNeverBrighter`、`TestDimTruecolorOut`、`TestDimDefaultFg` |

## components §圖與間距、§折行、§大寫

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| §圖與間距 | 圖的結構是規定（框、glyph 位置、間距、對齊、標的顏色）；「縮 N 格」是離框 N 格 | 每個畫面對照規範的圖 | 各元件的版面測試 |
| §圖與間距 | 欄與欄之間兩格；glyph 與它後面的字之間一格 | 表頭、表單、menu、radio、checkbox、file-picker | `TestTwoCellColumnGap`、`TestGlyphOneCellGap` |
| §折行 | textarea、viewer、confirm、toast、error popup、splash 太窄時都照同一套：在空白處斷，下一個字放不下就整個接到下一列；寬字每個字之間都可以斷，一列填滿再換列；一個字比一列還長才照格子切；寬字不切成兩半；斷在空白處時，textarea 與 viewer 的每個字元都要畫出來（一個字剛好填滿一列、後面接著空白，那個空白畫在下一列的第一格），只是顯示的文字（confirm、toast、error popup、splash）不畫那個空白 | I 的 `Comment`；D「Note viewer」把視窗縮窄、「Confirm: long」、「Error popup: long」；P 的 `Toast long`；把視窗縮窄按 `V` | `TestWrap`、`TestFoldAt`、`TestTextareaWrapAndNumbers`、`TestViewerFoldsAtSpaces`、`TestTextareaUpToShorterRow` |
| §大寫 | 要大寫：panel 膠囊與 tab、畫面 chip、popup 標題、表頭、清單的分組標題、一列一個值的 label、menu 與 key reference 的區塊標題；熱鍵標記裡的字母照原樣；名稱、路徑在標題與 hint 裡都照原樣（表單按鈕的 hint 只有動詞小寫：`Enter:export hosts.yaml`） | 每個畫面；畫面 chip `[t]ERMINAL`（小寫的熱鍵照原樣）；T terminal popup 的標題 `SESSION: shell`；D「Form: long」的 `EDIT: prod-db-01`；D「Form: with actions」的 `Export hosts.yaml` | `TestUppercaseWhereRequired`、`TestUppercaseKeepsHotkeyAndNames`、`TestPopupTitleSubject`、`TestFormButtonSubject` |
| §大寫 | 不改：值、menu 的 label 與說明、hint 與 footer 的說明、按鍵名、模式名、app 的內容 | 同上 | `TestUppercaseLeavesContent` |

## layout/screen

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| screen §由上往下 | 最上面一列（多畫面：chip 列 + 全寬分隔線；單畫面：可有 statusbar）、panel 區、最下面一列 footer；每個畫面的 panel 從 `[1]` 編號 | 每個畫面；lib 的單畫面版本由測試畫 | `TestScreenRows` |
| screen §footer | 一列、M5 寫法、鍵 Blue 冒號說明 Overlay0；放不下從尾端整組捨 | 縮到 40 欄 | `TestFooterDropsWholePairs` |
| screen §footer | 一般 `Space:menu ?:help Tab\|1–N:panels q:quit`；單 panel `Space:menu ?:help q:quit`；模式 `?:help` + 模式的鍵 + `Esc:<動詞>`；filter 打字中 `Enter\|Tab:list Esc:clear`，沒有結果時 `Tab:list Esc:clear`；PTY 在 panel 裡：出口鍵 + 保留鍵；popup 開著時照底下畫面的寫、跟著淡化 | P 一般與 Drag；P filter（打 `zz` 沒有結果）；T `[1] SHELL`；任一 popup；lib 的單 panel 版本 | `TestFooterVariants`、`TestFooterUnderPopupDimmed` |
| screen §footer | terminal popup 開著時 footer 整列不畫：最後一列兩端是空白，不露出 footer 的字 | T 的 terminal popup | `TestTerminalPopupSize` |
| screen §畫面 chip 列 | 每個畫面一個膠囊、串的顏色 Blue（選中填 Blue 字 Base 粗體、沒選中字 Blue）；放不下先縮 `[P] [I] [D] [T]` 再收掉沒選中的 | 每個畫面；縮窄 | `TestScreenChips`、`TestScreenChipsShrink` |
| screen §畫面 chip 列 | 右邊放 statusbar；下面全寬粗線 `━` Overlay0；長工作時從左填 Green 進度條、狀態文字 Green；terminal popup 開著時 chip 列與分隔線留著 | `Start task`；開 terminal popup 時再看一次 | `TestChipRowProgress` |
| screen §statusbar | 不是 surface；label 有熱鍵就標；一組 `label: 值`、冒號後空一格，組與組之間兩格、不加分隔符號（展示場：app 畫的）；一般 Overlay0、你在哪 Lavender、進行中 Green、錯誤警告數 Red / Peach；寬度固定、截斷不折行 | statusbar：`[N]arrow: off  icon: 1  powerline: 1  errors: 3  warnings: 2`（有熱鍵、值 Lavender）、`[S]tart task` 進行中 `copying: 4%` Green | `TestStatusbar` |
| screen §窄寬 | 放不下時只畫 focus 的 panel 佔滿寬、`Tab` 照換；門檻 app 定、不高於 80；窄寬只在最低尺寸以上才有 | `[N]arrow layout`；或縮到 59 欄以下 | `TestScreenNarrow` |
| screen §zoom | 放大一個 panel 是 app 的功能；預設跟窄寬一樣只畫 focus 的那一個、`Tab`、數字鍵照換、換到的也是放大的（lib：`App.ForceNarrow` 就是這個預設） | `[N]arrow layout` | `TestScreenNarrow` |
| screen §空狀態 | 置中一句事實 + 一句點名按鍵的提示，都 Overlay0，提示的鍵加方括號；panel 的 loading 與錯誤也用這個樣子；太長在空白處折行、每列置中、左右離框一格，不截斷 | P `[1] PODS` 的 `Empty`（把視窗縮窄） | `TestEmptyState`、`TestValuesEmptyState`（一列一個值的 panel：`Values.EmptyFact`、`EmptyHint`） |

## layout/panel

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| panel §框 | focus：Blue 粗線 `┏━┓`；失焦：Surface2 細直角線 `┌─┐`；同寬不位移；失焦內容淡化（串流除外） | `Tab` 換 panel | `TestPanelFrameFocus` |
| panel §框 | 上框左邊 `[N] label` 膠囊緊接框角；`N` 是跳到該 panel 的數字鍵；有 tab 時 `[N]` 自己一段、第一個 tab 的名字就是 label，在第一個 tab 時 `[N]` 與它之間一個空格（`[2] PODS`）；loading icon 放整串膠囊後、前後各空一格 | P `[1] PODS` 的 tab；`Loading` | `TestPanelTitle`、`TestPanelTabChain`、`TestPanelLoadingIconAfterChain` |
| panel §框 | 上框右邊模式名，夾在接頭之間（粗線 `┫ ┣`、細線 `┤ ├`），模式色加粗、大小寫照原樣；盡量一個詞；放不下先截標題；膠囊跟著換模式色 | P `[1] PODS` 的 `[D]rag`（粗線）；P `[2] LOG` 的 `[F]reeze` 之後按 `Tab`（細線）；縮窄 | `TestPanelModeLabel`、`TestPanelModeLabelKeptWhenNarrow` |
| panel §框 | 下框左邊 hint（每個 panel 都有；focus 在 panel filter 打字列上、在 panel 裡的 PTY 時不寫，回來時再寫）；下框右邊捲動位置 | 每個 panel；P 按 `/` 打字；T `[1] SHELL` 按 `Enter` 進去 | `TestPanelHint`、`TestPanelNoHintWhileFiltering`、`TestK10ExitKeyShown` |
| panel §框 | 框上的字與角之間隔一段線（`┗━ Enter:logs`、`3 of 40 ━┛`、`┫Drag┣━┓`），只有 `[N] label` 膠囊緊接左上角 | 每個 panel | `TestPanelFrameTextSpacing` |
| panel §框 | hint：這個 panel 自己的鍵（`Enter` 做什麼、常用熱鍵），不寫移動鍵與 footer 已有的 core key；清單一項都沒有時（panel filter 篩到空，或本來就空）拿掉 `Enter` 與對項目的熱鍵，只留對整個 panel 的；focus 時鍵 Blue 說明 Overlay0、失焦跟著淡化；放不下從尾端整組捨、捲動位置留著 | 每個 panel；縮窄；I `[2] FILES` 的 `Empty`（只剩 `/:find`），或按 `/` 打 `zz` 再 `Tab` | `TestPanelHintContent`、`TestPanelHintDrop`、`TestPanelHintNoItems` |
| panel §膠囊串 | 兩端斜角 U+E0BA / U+E0B0；選中的填串色、字 Base 粗體；沒選中的不填、字用串色；斜角與接縫照 Powerline 上色（前景左段的底、背景右段的底，沒選中的底當 Base）；串色：panel 是框線色、popup 是層色、chip 是 Blue；`[N]` 膠囊與目前 tab 都算選中 | P `[1] PODS` 換 tab；D「Popup with tabs」 | `TestCapsuleChain` |
| panel §框 | label 裡的名稱照原樣，跟 popup 的標題一樣：`[2] PODS: nginx-7d4f`；整個 label 就是名稱時照原樣（lib：`Panel.Name`；Space menu 與 `?` 的標題也一樣） | K `[2] KIND: pods` | `TestPanelName` |
| panel §膠囊串 | `[N]` 與第一個 tab 都選中時中間的空白跟接縫一樣佔一個 powerline 符號的寬度，換 tab 時字不位移 | P `[1] PODS` 換 tab（`SHOWCASE__POWERLINE_WIDTH=2` 也試一次） | `TestPanelTabSpaceIsAJoin` |
| panel §膠囊串 | 接縫：換色 U+E0B0、同色 U+E0B1；字前後不加空白、大寫；放不下先縮沒選中的成第一個字，再從尾端收成 `…`，選中的永遠留著、不撐寬框；選中的那一段自己就太長時字從尾端截加 `…`，兩端斜角照畫、跟框角之間留一段線 | 縮窄 | `TestCapsuleChainShrink`、`TestPanelModeLabelKeptWhenNarrow` |
| panel §一般清單 | cursor：Subtext1 底、Base 粗體、佔滿內寬、整列 Base 字；右邊的留白（跟箭頭一樣寬）畫 U+E0B0、Subtext1；失焦跟著淡化 | P `[1] PODS` | `TestListCursor`、`TestCursorArrow` |
| panel §一般清單 | 表頭 Blue 不加粗、大寫、固定不捲；每欄是數字或文字：數字靠右照數值排、文字靠左照文字排；欄間兩格；排序欄位加 U+F160（升）/ U+F161（降）；多欄排序在箭頭前加順位 `(2)` | P `[1] PODS` 的 `Sort by`（Name 升冪、Age 降冪、兩欄） | `TestListHeader`、`TestListSortGlyphs`、`TestListSortNumbers` |
| panel §一般清單 | 捲動位置：放不下時 `N of M`（框線色），全放得下不寫；沒有 cursor 的寫 `N-M of T` | P `[1] PODS`（40 列）、`[2] LOG` | `TestPanelScrollPosition` |
| panel §一般清單 | 標記欄：每列最前一格，標記的畫 Green U+F0132，沒標記的空白，欄寬一直保留 | P `[1] PODS` 的 `[m]ark` | `TestListMarkColumn` |
| panel §一般清單 | lib：app 照 ID 設標記（`List.SetMarked`），篩選中沒顯示的列也勾得到 | P `[1] PODS` 的 `[m]ark` | `TestListMarkAndSelectByID` |
| panel §一般清單 | Drag mode 拿著的那一列（進 Drag 的鍵由 app 決定）：照 cursor 畫，Subtext1 換成 Yellow 底、Base 粗體，箭頭也是 Yellow；標記欄畫 U+F04E2，放下後回到原本的標記；有 Drag 的 panel 每一列都保留標記欄；按鍵同 reorder 拿著時（`j\|k` 搬、到頭尾繞回，`Enter` 放下，`Esc` 放回）。lib：`List.Draggable`、`Hold`、`MoveHeld`、`Drop`，Drag mode 與它的鍵由 app 登記 | P `[1] PODS` 的 `[D]rag`（先 `m` 標記一列再拿起它） | `TestPanelDragHeldRow` |
| panel §一般清單 | 截斷從尾端加 `…`；路徑、檔名可以從前面截；兩端都有意義的從中間截（`demo-app…66-gm72n`），從哪裡截由 app 替每一欄決定（lib：`Column.Path`、`Column.Middle`） | P `[1] PODS` 的長名字（從中間截）；I `[2] FILES` 的路徑 | `TestListTruncate`、`TestTruncateMiddle` |
| panel §一般清單 | 欄放不下時先整欄收（表頭一起），照 app 標的順序，名稱欄不收；還放不下才照比例縮；變大時照順序回來（lib：`Column.Drop`） | P `[1] PODS` 把視窗縮窄：`AGE`、`RESTARTS`、`READY` 依序收掉 | `TestListColumnsGiveWay` |
| panel §一般清單 | 單欄的清單沒有欄名時不畫表頭 | K `[1] KINDS` | `TestListColumnsGiveWay` |
| panel §位置列 | 清單上面一列寫目前的位置（要不要由 app 決定）：panel 內第一列、離框一格，在 panel filter 的打字列與表頭上面，固定不捲；Lavender、失焦跟著淡化；太長從前面截；只顯示：cursor 不停、捲動位置不算（lib：`List.Location`）；清單空了時照樣留著，空狀態只取代表頭與清單（G98 定案 A） | I `[2] FILES`（假的專案的目錄）；按 `/` 打字列在它下面 | `TestListLocationRow` |
| panel §一般清單 | 排序的群：app 替每一列指定群、群的先後由 app 定，排序只在群裡，群之間不畫標題（lib：`Row.SortGroup`） | I `[2] FILES`：資料夾裡的檔在前、最上層的在後，各自照名稱排 | `TestListSortGroups` |
| panel §一般清單 | 清單的順序要固定：同一份資料每次排出來都一樣；沒有順序的資料先排（lib：`List.SortBy` 照名稱、數字照數值，排不出先後的照 `Row.ID`） | P `[1] PODS`、I `[2] FILES` 一打開就照名稱排 | `TestListFixedOrder` |
| panel §一般清單 | 空的照空狀態 | `Empty` | `TestEmptyState` |
| panel §分組 | 分組標題 Blue、大寫、不加粗、沒有 glyph、離框一格；項目多縮兩格；組與組之間不空列、不畫分隔線；失焦跟著淡化 | K `[1] KINDS` | `TestPanelListGroups`、`TestPanelDetailsFollowCursor`（失焦） |
| panel §分組 | 有標記欄時，標記畫在多縮的那兩格裡：標記離框一格、空一格，字離框三格；分組標題那一列不留標記欄 | 展示場的分組清單沒有標記（看得到的是 P `[1] PODS` 的一般清單） | `TestPanelListGroupsMarks` |
| panel §分組 | cursor 跳過分組標題：`j\|k` 越過它、`gg\|G` 到第一個與最後一個項目、`N of M` 只算項目；捲到一組的第一個項目時，這組的標題也留在畫面裡 | K `[1] KINDS`（把終端機拉矮再捲） | `TestPanelListGroupsCursor` |
| panel §分組 | 沒有項目的組不畫標題（panel filter 篩掉的組、app 給的空組） | K `[1] KINDS` 按 `/` 打 `sec` | `TestPanelListGroupsFilter` |
| panel §loading 與錯誤 | loading：icon 在整串膠囊後、用框線色（focus Blue、失焦 Surface2）；沒內容時中間寫一句（`󰪞 loading pods…`）；重新載入舊內容留著 | `Loading`（空的時候與有內容時各一次） | `TestPanelLoading`、`TestPanelLoadingIconAfterChain`（顏色） |
| panel §loading 與錯誤 | 錯誤：置中兩句，事實 Red、提示 Overlay0；舊內容在就留著、錯誤用 toast 報 | `Fail`（空的時候與有內容時各一次） | `TestPanelError` |
| panel §一列一個值 | 照表單：項目逐一移動（有 tab 時 `h\|l` 給 tab）；`Enter` 開 input popup，radio、checkbox 原地選；確定後 focus 留在同一列 | P `[3] SETTINGS`；I `[1] INPUTS` | `TestValuePanelMoves`、`TestValuePanelEnter` |
| panel §一列一個值 | 沒有送出：確定的那一刻就生效；沒有動作區、沒有「改過了要先問」 | 同上 | `TestValuePanelAppliesNow` |
| panel §明細 | 另一個 panel 的 cursor 一移動，明細立刻換成那一筆，label 寫那一筆的名字時一起換；不用 `Enter`，cursor 移動時 focus 不動；app 也可以讓清單上的 `Enter` 把 focus 送到明細（K3） | K `[1] KINDS` 上 `j\|k`，看 `[2]`；`Enter` 進 `[2]` | `TestPanelDetailsFollowCursor`、`TestValuesOnMove`、`TestFilePreviewPanel`、`TestFilePreviewPanelHidesSecrets`（預覽當明細：`FilePreview`） |
| panel §一列一個值 | focus 色塊 Subtext1 底 Base 粗體、最右邊一格是箭頭，從項目開頭到 panel 內右緣；label 大寫，一般 Blue、停用 Surface2、出錯 Red；label 欄寬取最長的 label、與值之間兩格 | `[3] SETTINGS` 的 `Disable`（`Remote` 停用）、`Fail`（`Timeout` 的 label 變 Red） | `TestValuePanelFocusBlock`、`TestValuePanelLabels` |
| panel §一列一個值 | 捲動位置照表單以欄位為單位：radio 的幾個選項算同一欄 | P `[3] SETTINGS` 縮矮 | `TestValuePanelScrollByField` |
| panel §一列一個值 | `?` 最前面可以有「每一列」一節（`what each row is`）：設定的名稱用一般字色（Text）、右欄說明，按鍵的節接在後面；要不要有由 app 決定（lib：`Field.Desc`，有一欄有就有這一節） | P `[3] SETTINGS` 按 `?` | `TestKeyReferenceWhatEachRowIs` |
| panel §一列一個值 | 表頭（`PROPERTY  VALUE`）Overlay0，不用 label 的 Blue；panel 失焦時跟著淡化（T2）；一般清單的表頭照舊 Blue | P `[3] SETTINGS` 的 `PROPERTY  VALUE`；I `[1] INPUTS` 的 `INPUT  VALUE` | `TestValuePanelHeader` |
| panel §panel filter | `/` 在最上面出現打字列（清單有表頭時在表頭上面）、即時篩選、不開 popup；最前面 U+F105 離框一格（跟 finder 的打字列一樣）、空一格才接打的字，打字時框線色（Blue）、在清單時 Overlay0；右邊 `符合/總數`，Overlay0；打字列按鍵與比對同 finder；篩選後保持 panel 的排序 | P `[1] PODS` 按 `/` | `TestPanelFilter` |
| panel §panel filter | 清單上 `Tab` 下一個 panel；清單上 `Esc` 清掉篩選；選了之後清單留著篩選、打字列變灰；清單上 `/` 回打字列接著打 | 同上 | `TestPanelFilterListKeys` |
| panel §panel filter | focus 離開篩選留著；回來時直接在打字列上（`Tab` 經過停兩下） | 同上 + `Tab` 繞一圈 | `TestPanelFilterFocusReturn` |

## layout/popup

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| popup §外框 | 粗線 `┏━┓`、層色；被蓋在底下照樣是粗線 | 每種 popup | `TestPopupFrame` |
| popup §外框 | 標題：glyph + 文字做成膠囊，放在 `┏━` 後；兩端圓角 U+E0B6 / U+E0B4；填層色，字 Base 粗體、大寫（名稱與路徑照原樣）；寫對哪一個做時，名稱接在 `: ` 後、照原樣（viewer 的 ` — `、file-picker 的 ` · ` 照它們自己的）；太長從尾端截加 `…` | 每種 popup；D「Popup: long title」；D「Confirm」`DELETE: web-a8k2`、「Form: long」`EDIT: prod-db-01`；I `Name` 按 `Enter` 的 `RENAME: assets` | `TestPopupTitleCapsule`、`TestPopupTitleTruncate`、`TestPopupTitleSubject`、`TestTruncateKeepsStyle`（截掉時 `…` 跟著被截那段的樣式）、`TestPopupSubjectErrorMenu`（`App.MenuAbout`：`OPEN WITH: notes.md`） |
| popup §外框 | 標題後接 tab 照膠囊串畫（跟圓角的標題之間隔一段線） | D「Popup with tabs」 | `TestPopupTitleTabs` |
| popup §外框 | 標題 glyph：menu U+F0C9、key reference U+F059、confirm U+F0ECC、toast 依種類、terminal U+F018D、input／表單／note 由 app 選 | 每種 popup | `TestPopupTitleGlyph` |
| popup §外框 | hint 放 `┗━ ` 後；順序 `Enter` → 自己的鍵 → `Tab:…` → `Esc`（terminal 出口鍵第一）；不寫移動鍵（換塊的照寫，表單的 `Tab:actions`）；框到最寬還放不下才從尾端整組捨（terminal 太窄時），但最後的 `Esc` 一定留著，從它前面那一組開始捨；只有打字列的列全部、80 欄放得下 | 每種 popup；把 terminal 縮到很窄 | `TestPopupHintOrder`、`TestPopupHintDrop` |
| popup §外框 | 框上的字與角之間隔一段線：`┏━TITLE`、`┗━ Enter:run`、`4 of 12 ━┛` | 每種 popup | `TestPopupFrameTextSpacing` |
| popup §外框 | 內容上下各留一列、左右離框一格（terminal 上下左右都不留） | D「Menu: long」 | `TestPopupMargins` |
| popup §外框 | 清單的 cursor 從框貼到框、字照樣離框一格；表單的 focus 色塊從值到框；右邊的留白是 U+E0B0、色塊的顏色，跟箭頭一樣寬；只有整列的 cursor 有箭頭 | Space menu；D「Form: New host」 | `TestPopupCursorCoversPadding`、`TestCursorArrow` |
| popup §外框 | 分隔線不接框：橫的左右各留一格、直的不畫進上下留白；Overlay0 | Space menu（區塊之間）；D「Menu: Open with」；I 的 `Meeting` | `TestPopupSeparator` |
| popup §寬度 | 兩種寬度（見 F7）；「確定」= 打開那一刻就知道最寬（PIN 一開就 43 格：hint 37 + 6）；要捲的也放得下捲動位置（hint + 捲動位置最寬時 + 9，`20 of 20`、`T-T of T`），捲的時候框不變寬窄；select、checkbox popup 也是（demou 的 DM3）；篩到沒有結果時 hint 變短，寬度照樣是打開時的 | I 的 `PIN`；D「Menu: long」、「Confirm: long」在矮的 terminal | `TestF7WidthKnown`、`TestPopupWidthCountsScroll`、`TestFinderHintNoResults` |
| popup §寬度 | 窄的內容靠左離框一格；toast 文字與 password 解鎖遮罩置中 | D「Confirm」、「Toast」；I 的 `PIN` | `TestPopupNarrowContentAlign` |
| popup §開關動畫 | 拉線 4 格（正中間那一列的層色 `━` 從中間往左右拉到 popup 寬）→ 停 1 格 → 展開 4 格（從中間往上下露出）；收起倒著播；每格 32ms | 每種 popup，含 toast 與 terminal | `TestF2AnimationFrames` |
| popup §開關動畫 | 打開的第一格起底下就淡化；開始收起時底下立刻亮回來；打開的動畫播完前與收起時不收鍵 | 同上 | `TestF2DimFromFirstFrame`、`TestF2NoKeysWhileOpening` |
| popup §loading | icon U+F0A9E–U+F0AA5 八格；一格 90ms；格數由時鐘決定 `frames[(now / 90ms) % 8]`；只在有東西 loading 時續排 tick；一格寬、換上換下不位移；顏色跟旁邊字、popup 標題用層色粗體、panel 上框用框線色；放在整串 tab 後、前後各空一格 | D「Loading popup」；P `Loading` | `TestLoadingIconFrames`、`TestLoadingTickOnlyWhenLoading` |
| popup §取消與完成 | 取消回 source；完成照 T1 | D 的疊法 | `TestF4KeepSource`、`TestT1KeepOrClear` |
| popup §內容放不下時 | 只有內容區捲，留白、錯誤列、上下框固定；跟著 focus 捲最少的量；`N of M`（有 cursor）／`N-M of T`（note），層色；有游標的（textarea、選取模式裡的 viewer）也寫 `N of M`，以行為單位；表單以欄位為單位、只算欄位，focus 在錯誤列或按鈕上時不寫 | D「Menu: long」、「Note viewer」、「Form: long」 | `TestPopupScroll`、`TestPopupScrollPosition`、`TestFormScrollPosition`、`TestInputShortScreenKeepsTypingRow`（輸入框太矮時打字列與錯誤列留著）、`TestAppPopupScroll`（app 自己的 popup：body 的 `Scroll()`） |
| popup §錯誤列 | 內容區下面、空一列；只有值可能不合格的預留；只放值的錯；Red、左緣對齊（內容置中時也置中）；框與標題維持層色 | I 的 `Port` | `TestErrorRow` |
| popup §錯誤列 | 只佔一列、截斷加 `…`；表單上可 focus、`Enter` 開完整訊息；input popup 裡不拿 focus；只在檢查時改 | D「Form: New host」送出失敗；I 的 `Port` | `TestErrorRowTruncate`、`TestErrorRowChangesOnCheckOnly` |
| popup §錯誤列 | 值錯太多次而暫停收值時倒數寫在錯誤列：Red、每秒更新（`try again in 27 s`）、倒數完清掉；暫停期間除了 `Esc` 都吞掉（lib：`PasswordInput.PausedUntil`） | I 的 `Unlock` 連錯三次 | `TestPasswordPause` |
| popup §錯誤列 | 動作失敗開 error popup 蓋在上面，關掉回原本的 popup、值都在 | D「Form: save fails」 | `TestActionFailureErrorPopup`、`TestCommitFailureKeepsPopup`（input 寫回失敗：`Commit`） |

## dialog/form

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| form §用途 | 表單上不打字：要打字或選項放不下的欄位 `Enter` 開 input popup；radio、checkbox 畫在表單上原地選 | D「Form: New host」 | `TestFormNoTyping` |
| form §標題 | 寫表單要做什麼（標題膠囊），不寫型別；改的是某一筆時可以帶上它的名稱，`: ` 後照原樣 | 同上；D「Form: long」`EDIT: prod-db-01` | `TestPopupTitleSubject` |
| form §欄位的排法 | 上下式、左右式，可混用；欄位之間、label 與值之間不空行；label、值、錯誤左緣對齊；左右式 label 欄寬取左右式最長的、與值之間兩格 | D「Form: New profile」（混用） | `TestFormLayouts` |
| form §選項畫在表單上 | 五個以內畫在表單、超過用 select／checkbox popup；radio 與 checkbox 組一律直排、一個選項一列 | D「Form: New host」的 `Auth`、`Order` | `TestFormOptionsDrawn` |
| form §focus 與 label | focus 照 menu 的 cursor（層色底 Base 粗體），從項目開頭一直到框（蓋住右邊留白）、最右邊一格是箭頭；按鈕的色塊就是按鈕、沒有箭頭；值空著也一整塊；箭頭兩格時放得下的值照樣整個畫、不加 `…`；每一列右邊的留白跟箭頭一樣寬（兩格時留兩格） | 同上；`SHOWCASE__POWERLINE_WIDTH=2` | `TestFormFocusBlock`、`TestFormFocusWideArrow`、`TestFormRightPadIsArrow` |
| form §focus 與 label | label 不因 focus 變色：一般層色、停用 Surface2、值出錯 Red（優先）；停用的留原位；值不跟 label 同色 | D「Form: New host」的 `Credential`（停用）；送出失敗 | `TestFormLabelColors` |
| form §動作區 | 按鈕：文字左右各一格、一塊底色；沒 focus Surface1 底 Text 字、focus 層色底 Base 粗體；寫按下去要做的事；不用 `[ Save ]`、不用膠囊 | D「Form: New host」 | `TestFormButton` |
| form §動作區 | 位置：欄位區 → 空一列 → 錯誤列 → 分隔線 → 動作區；按鈕靠右離框一格；錯誤列、分隔線、動作區固定不捲；沒有取消按鈕 | D「Form: long」（欄位要捲） | `TestFormActionRowFixed` |
| form §動作區 | 按鈕可以不只一個：主要的送出在右下角；每一列放哪些按鈕、排幾列由 app 決定，每一列靠右、同一列的按鈕之間隔一格。lib：`Form.Buttons`（一列一列，主要的送出接在最後一列的最後；最後一列空著就是它自己一列） | D「Form: with actions」（`Test` `Scan` 一列、`Save As` `Save` 一列） | `TestFormButtonsLayout` |
| form §動作區 | 其他按鈕：另一種送出（`FormButton.Submit`，整張表單檢查、送出後關掉）或不關表單的動作（`FormButton.Run`）；兩個都給或都不給是程式的錯 | D「Form: with actions」的 `Save As`、`Test` | `TestFormSaveAs`、`TestFormButtonNeedsOne` |
| form §動作區 | 不關表單的動作：先只檢查它要的欄位（`Needs`），沒過跟送出被擋一樣；跑的時候標題後面放 loading icon；成功 toast 一句、結果是內容時開 note、失敗開 error popup（關掉後 focus 在按鈕上，再按 `Enter` 重試）；focus 留在按鈕上；動作寫進某一欄時算改過（`FormResult.Apply`） | D「Form: with actions」：Host 空著按 `Test`；填了再按（等一下才連上）；Host 打 `bad-1` 再按；`Scan` 開 note、把 Port 寫成 22，之後 `Esc` 先問 | `TestFormActionRun`、`TestFormActionResults` |
| form §動作區 | 還在跑時：focus 停在那顆按鈕上按 `Enter` 不作用，hint 拿掉它的 `Enter`，跑完放回來；跑得久的送出也一樣（lib：`SubmitLong`，失敗開 error popup、完成關掉表單） | D「Form: with actions」按 `Test` 後馬上再按；主要的送出是跑得久的 `Connect` | `TestFormActionWhileRunning`、`TestFormSubmitLong` |
| form §動作區 | 還沒跑完表單就關掉（`Esc`、按了另一顆送出）：動作算取消、能停就停（`Run` 與 `SubmitLong` 拿到的 ctx 結束），結果不顯示：沒有 toast、note、error popup，也不做 `Apply` | D「Form: with actions」按 `Test` 後馬上 `Esc` | `TestFormClosedDropsResult` |
| form §移動 | 項目：欄位、每個選項、每個按鈕、有錯誤的錯誤列；分兩區：欄位區是欄位與錯誤列（錯誤列在最後一欄後面），動作區是按鈕 | D「Form: New host」 | `TestFormItems` |
| form §移動 | `j\|l\|↓\|→` 下一個、`h\|k\|↑\|←` 上一個，只在 focus 所在的那一區；跨欄；動作區照閱讀順序（一列由左到右、再下一列）；`gg\|G` 這一區的頭尾；`u\|d` 以項目為單位半頁、到頭尾停住；每一區各自頭尾繞回 | 同上；D「Form: with actions」的按鈕 | `TestFormMoveItems`、`TestFormWrap`、`TestFormHalfPage`、`TestFormActionAreaMoves` |
| form §移動 | `Tab\|Shift-Tab` 在欄位區與動作區之間來回，兩邊都回到離開前那一項；第一次到動作區停在主要的送出；錯誤列屬於欄位區（在上面按 `Tab` 到動作區）、`j\|k` 與 `e` 到得了 | D「Form: long」改一欄後 `Tab`、`Enter`；D「Form: with actions」 | `TestFormTabBlocks`、`TestFormActionAreaMoves` |
| form §移動 | 停用的跳過；打開時 focus 在第一個能停的（從某一欄打開就停那欄）；不自動打開第一欄；跳進 radio、checkbox 那一欄（從那欄打開、送出被擋、`e` 回去）停在第一個有值的選項（radio 已選的、checkbox 第一個勾的，都沒有停第一個） | D「Form: open at Port」；停用的選項：D「Form: New host」的 `Order` 的 `random` | `TestFormSkipDisabled`、`TestFormSkipDisabledOption`、`TestFormOpenFocus` |
| form §`Enter` | 欄位：開 input popup 疊在表單上，確定寫回 focus 留原欄，`Esc` 值不變；checkbox popup：`Space` 勾、`Enter` 寫回、`Esc` 取消 | 同上的 `Name`、`Namespaces` | `TestFormEnterField` |
| form §`Enter` | radio 選項：選它；checkbox 選項：原地翻（`Space` 也能翻）；都不開 popup、focus 不動、馬上生效；按鈕：按下它（送出或做它的動作） | 同上 | `TestFormEnterOption`、`TestFormEnterButton`、`TestFormCheckboxSpace` |
| form §送出 | 只有送出按鈕的 `Enter`；`Tab` 第一次就停在主要的送出；不用字母熱鍵；`Esc` 不把 focus 移到動作區 | 同上 | `TestFormSubmitOnlyButton` |
| form §在欄位上打字 | 要打字的欄位上按表單操作鍵以外的可列印字元（含數字、`q`）跳說明 note：標題是欄位名、`Enter`／`Esc` 關、focus 不動；其他項目上不跳 | D「Form: New host」的 `Name` 上打 `s` | `TestFormTypingNote` |
| form §在欄位上打字 | note 只收 `Enter`、`Esc`：其他可以打的字（含 `q`、`?`、app 的 core key）都接住、不作用，`Ctrl-C` 照樣有效（K9）；很窄時折行、不截斷 | 同上，接著打 `q` `?`；把視窗縮窄 | `TestFormTypingNote` |
| form §動作區 | 按鈕能在表單上開 popup（例：terminal popup 跑 `ssh-copy-id`），表單留在底下；那個 popup 關掉、回到表單時 app 重讀表單的值，算改過（lib：`FormButton.Open`、`Back`） | D「Form: with actions」的 `Copy key`（跑完 Auth 變成 privatekey） | `TestFormButtonOpens` |
| form §在欄位上打字 | 表單操作鍵不跳 note：移動鍵、`u\|d`、`g`（單按不作用）、`G`、`e`（沒錯時不作用）、`Enter`、`Esc`、`Space`（不作用）、`?`（key reference） | 同上，按 `e` `u` `g` `?` | `TestFormOperationKeys` |
| form §hint | `Enter:` 後的字跟著 focus 換（`edit`／`choose`／checkbox 選項 `Enter\|Space:toggle`／按鈕小寫）；有問題的欄位加 `e:error`；`Tab` 寫要去的那塊：欄位區（含錯誤列）`Tab:actions`、按鈕上 `Tab:fields`；錯誤列 `Enter:show e:edit Tab:actions Esc:cancel`；按鈕寫按鈕的字，只有動詞小寫，名稱、路徑照原樣（`Enter:save as`、`Enter:test`、`Enter:export hosts.yaml`；lib：`FormButton.Subject`、`Form.ButtonSubject`，失敗的 error popup 標題名稱接在 `: ` 後照原樣）；不寫 `j\|k`；`Esc:cancel` | 同上 | `TestFormHint`、`TestFormButtonSubject` |
| form §取消 | 沒改過 `Esc` 直接關；改過先 confirm（`Enter` 放棄、`Esc` 回表單值都在）；「改過了」只要寫回過就設、不清掉；只有表單、textarea 與 reorder 問 | D「Form: New host」改一欄再 `Esc`；I 的 `Segments` | `TestFormDiscardConfirm`、`TestFormDirtyFlag`、`TestReorderEsc` |
| form §錯誤 | 一欄自己的規則在 input popup 擋；整張表單的規則送出時檢查；沒過：不存、每個有問題的 label Red、錯誤列寫第一個、focus 跳過去 | D「Form: New host」空著送出、Port 打 `0` | `TestFormValidation` |
| form §錯誤 | `e`：有問題的欄位 → 錯誤列換成這一欄的訊息；錯誤列 `Enter` 開完整訊息的 error popup（標題欄位名、`Enter`／`Esc` 關）；錯誤列 `e` 回那一欄 | 同上 | `TestFormSubmitChecks` |
| form §錯誤 | 送出動作失敗：error popup，關掉回表單、focus 在按鈕；錯誤跟著檢查走（送出與不關表單的動作都算檢查，直到下一次檢查）、第一次送出前不檢查 | D「Form: save fails」 | `TestFormSaveFails`、`TestFormNoCheckBeforeSubmit` |

## dialog/menu

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| menu §長相 | label Text、`[x]` 同色；說明 Overlay0 靠右離右框一格、太長截加 `…`；label 與說明之間最少兩格 | Space menu；D「Menu: long descriptions」 | `TestMenuRowLayout` |
| menu §長相 | 區塊標題：U+F02D8（六角形）+ 一格 + 大寫標題，Overlay0、離框一格、不加粗、cursor 跳過；區塊之間的分隔線 `╌`（U+254C）、不接框、Overlay0、cursor 跳過 | Space menu | `TestMenuHeadings`、`TestMenuSeparator` |
| menu §長相 | cursor 層色底、Base 粗體，從框貼到框、字離框一格、最右邊一格是箭頭；停用的列整列 Surface2、cursor 跳過、熱鍵不作用 | `Disable` 之後的 Space menu | `TestMenuCursor`、`TestM6DisabledRow` |
| menu §長相 | 標題 U+F0C9 + 文字（標題膠囊）：Space menu 寫 focus panel 的 `[N] label`、其他寫它做什麼（`GLOBAL OPERATION`、`SORT BY`）；`Global operation` 列說明一律 `actions for the whole app`；寬度照「內容確定」 | Space menu、Global operation、Sort by | `TestMenuTitle`、`TestMenuGlobalRow` |
| menu §工作清單 | 列跟著工作更新（menu 開著時進度、狀態一變那一列就變；lib：`Action.Live`，列裡的 loading 圖示 `LoadingFrame`、`App.Spinning`）；能對那一列做事的 `Enter` 開帶全文的 confirm、hint 寫那個動作 `Enter:cancel`（lib：`Action.EnterHint`）；沒有動作的 `Enter:detail`；hint 跟著 cursor 那一列換 | D「Menu: Worklist」的 `upload.tar` | `TestMenuLiveRows`、`TestSpinning` |
| menu §按鍵 | `j\|k` 到頭尾繞回；`u\|d`（到頭尾停住）、`gg\|G`；`Enter` 執行、熱鍵直接執行；`Esc` 關（global 回 Space menu）；Space menu 再按 `Space` 關；hint `Enter:run Esc:close`；打開那一列全文的 menu（工作清單）寫 `Enter:detail Esc:close` | 同上 | `TestMenuKeys`、`TestMenuWraps`、`TestMenuEnterOpensFullText` |

## dialog/confirm

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| confirm §長相 | 標題 U+F0ECC + 動作名（大寫）；問句放最後、下面接 hint；內容放問句上面、空一列 | D「Confirm」「Confirm: with details」 | `TestConfirmLayout` |
| confirm §長相 | 問句 Text 不加粗、內容 Overlay0、警告那一句 Peach（哪句由 app 定） | D「Confirm: with details」 | `TestConfirmColors` |
| confirm §長相 | 太長折行不截；內容長時只有內容 `j\|k` 捲、問句固定；寬度照「內容確定」 | D「Confirm: long」 | `TestConfirmWrapAndScroll` |
| confirm §按鍵 | `Enter` 接受、`Esc` 取消，沒有 `y`、`n`；`Space` 不作用；hint `Enter:<動詞> Esc:cancel` | D「Confirm」 | `TestConfirmKeys`、`TestConfirmHint` |

## dialog/note

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| note §長相 | 留白一律有；標題 glyph（app 選）+ 顯示的東西（標題膠囊，`YAML — pod/nginx`）；捲動位置 `N-M of T`（viewer 的選取模式裡 `N of M`）；寬度：確定的照內容、viewer 用上限 | D「Note viewer」 | `TestNoteLayout` |
| note §長相 | viewer 預設折行；要對齊的內容 app 可以讓它不折行：比框寬的截掉、行尾加 `…`（lib：`Viewer.NoWrap`）；選取模式的游標走進截掉的部分時，整個畫面跟著左右捲到看得見（像 vim 的 nowrap），左邊捲掉的開頭也加 `…`，截掉的照樣選得到、複製得到（G99 定案 A；terminal 的選取模式同一套）；畫不畫行號由 app 決定（`Viewer.NoNumbers`），不畫時字離框一格 | D「Viewer: a table」（按 `v` 再 `$`） | `TestViewerNoWrapNoNumbers`、`TestViewerNoWrapFollowsCursor` |
| note §長相 | viewer 的搜尋：命中的字全部加底線、保留原本的顏色；目前那一筆畫成 cursor 色塊；不另外加顏色（Yellow 只給選取）。lib：搜尋由 app 做（`/`、`n\|N`），把命中交給 `Viewer.Hits`、`Current`（`FindText` 找不分大小寫的） | D「Note viewer」按 `/` 打 `nginx`，再 `n`、`N` | `TestViewerSearchHits` |
| note §viewer 的行號 | viewer（YAML、App Log、檔案內容）左邊有行號：每行第一列、靠右、後空一格、折下來的列空白；Overlay0，選取模式裡游標那行 Text；至少三位：行數固定的取 max(3, 位數)、打開後不變，串流的超過 999 行才跳寬；複製不含行號；`N of M` 跟行號同一套數字 | D「Note viewer」；P `[2] LOG` 按 `Enter`；I `[2] FILES` 的 `[V]iew` | `TestViewerLineNumbers`、`TestViewerLineNumbersStream`、`TestViewerCopyNoNumbers`、`TestViewerFoldedScroll` |
| note §viewer 的行號 | 有 tab 的 viewer：行號欄照看過的頁裡最寬的，只變寬、不變窄（串流的頁也一樣） | D「Popup with tabs」、DevTools 切 tab | `TestViewerTabGutterOnlyGrows` |
| note §viewer 裡的 Tab 與控制字元 | Tab 畫成 `input_tab_width` 格空白、不對齊 tab 停點；換行前的 `\r`（CRLF）不畫；其他控制字元畫成名字、Red；複製照原樣；file-picker 預覽的文字檔一樣 | I `[2] FILES` 的 `main.go` 按 `[V]iew`（有 Tab）；`[/] Find` 預覽同一個檔 | `TestViewerTabs`、`TestViewerControlChars` |
| note §viewer 的語法顏色 | 有固定格式的照 file-picker 預覽上色（chroma catppuccin-mocha、只上前景色）；語言由 app 給（`Viewer.Lang`），檔案內容照檔名；沒有固定格式或判斷不出來的是純文字 Text（建議；app 自己給的顏色保留，例：log 的 ERROR、WARN 行）；行號 Overlay0；選取範圍 Yellow 底 Base 字、範圍裡不顯示語法顏色 | D「Note viewer」（YAML）；I `[2] FILES` 的 `[V]iew`；P `[2] LOG` 按 `Enter`（ERROR 紅、WARN 橘保留） | `TestViewerSyntaxColours` |
| note §按鍵 | 折行的 viewer 以畫出來的一列為單位捲（`j\|k` 一列、`u\|d` 照列算）；`N-M of T` 的 N、M 是最上面、最下面那一列所屬的行；選取模式照舊以行為單位 | D「Note viewer」把視窗縮窄 | `TestViewerScrollsRows` |
| note §按鍵 | 捲動 `j\|k` `u\|d` `gg\|G`；只用 `Esc` 關（`Enter`、`Space` 不關）；error popup（不管怎麼打開的）與表單的打字說明 `Enter` 也能關；hint 熱鍵在前、`Esc:close` 最後 | 同上；D「Error popup」 | `TestNoteKeys` |
| note §選取模式 | 每個 viewer 都有（lib：沒給 `Viewer.Select` 時是 `v`、`Visual`）；`y` 複製之後跳 Success toast；沒給 `Copy` 時 lib 複製到終端機的剪貼簿（OSC 52，跟著下一個畫面送出）並跳 `Copied` | D「Note viewer」按 `v` `v` `l` `y`；D「Popup with tabs」的 `Copy` 是 app 自己的 | `TestViewerSelectsByDefault`、`TestClipboardGoesWithTheNextFrame` |
| note §選取模式 | 只有 viewer 有；平常沒有游標（`j\|k` 捲、有 tab 時 `h\|l` 切 tab）；第一次 `v` 進模式（框與模式名 Yellow），游標在畫面第一行行首、還沒選，hint `v:select Esc:cancel`；`h\|j\|k\|l` `w\|b\|e` `0\|$` `u\|d` `gg\|G` 移游標、畫面跟著捲；第二次 `v` 從游標開始選（以字為單位、含游標那個字），hint `y:copy Esc:cancel`，再按 `v` 不作用；`y` 複製、整個離開；`Esc` 在哪一步都整個離開、不複製；游標層色底 Base 字，範圍 Yellow 底 Base 字、空行畫一格；捲動位置 `N of M`（游標的行） | D「Note viewer」按 `v` | `TestNoteHotkeysAndMode`、`TestViewerSelectScrollPosition` |
| note §key reference | 標題 U+F059 + `<surface> keys`（`[2] HOSTS KEYS`、`SPACE MENU KEYS`）；每列一個鍵：縮兩格、鍵補到最寬、空兩格、說明（截斷不折行）；從 menu 來的列 `label — 說明`、label 去掉標記、大小寫照原樣；標題裡的名稱照原樣（`FIRST PAGE — Pride and Prejudice KEYS`） | 每個 surface 按 `?` | `TestKeyReferenceLayout`、`TestKeyReferenceTitleKeepsName` |
| note §key reference | panel 上的區塊依序：`item operation`、`panel operation`、`move`（真的能用的移動鍵）、`core keys`（`Space`、`?`、`Tab`、`1–N`、`q\|Ctrl-C`；`Esc` 只在有作用時）、`app-wide`；區塊標題同 menu | P `[1] PODS`（有 tab：列 `h\|l`）、`[2] LOG`、有篩選時（列 `Esc`） | `TestKeyReferenceSections`、`TestKeyReferenceMoveSection`、`TestKeyReferenceCoreKeys` |
| note §key reference | popup 上只列它的熱鍵、`Enter`、`Esc`、移動鍵；鍵 Blue 不加粗、說明 Text、按不了的 Surface2；`?`／`Esc` 關、`Enter` 不作用；hint `?\|Esc:close` | Space menu 上按 `?` | `TestKeyReferenceKeys` |
| note §error popup | 框、文字 Red；標題膠囊填 Red、字 Base 粗體；標題 U+F0238 + 哪件事失敗（表單錯誤列打開的寫欄位名）；訊息折行、太長 `j\|k` 捲 | D「Error popup」「Error popup: long」（`j\|k` 捲）「Form: save fails」 | `TestErrorPopupLayout`、`TestPopupSubjectErrorMenu`（`App.ErrorAbout`：`RENAME FAILED: notes.md`） |
| note §error popup | `Enter`、`Esc` 都能關，hint `Enter\|Esc:close`；關掉回底下的 popup、focus 原位、值都在 | 同上 | `TestErrorPopupClose` |
| note §長相 | 一段話的 note（error popup、表單的打字說明）放不下時折行、不截斷；照它拿到的寬度折，終端機大小還不知道時打開的（`App.Start`）也在知道之後照實際的寬度折 | D「Error popup: long」把視窗縮窄 | `TestErrorPopupBeforeTheSize`、`TestFormTypingNote` |

## dialog/terminal

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| terminal §用途 | PTY 在 panel 裡時不是 popup：panel 的框、鍵寫在 footer；focus 在 PTY 裡時 panel 下框不寫 hint | T `[1] SHELL` | `TestK10KeysToChild`、`TestK10ExitKeyShown` |
| terminal §長相 | 大小照 F7 的例外（chip 列與分隔線留著）；沒有 statusbar 的 app 從第一列開始；上下左右都不留白；標題 U+F018D + 跑的是什麼；子程序的游標畫成一格 Text 底、Base 字的方塊，DECSCUSR 要的形狀不理（panel 裡也一樣） | T `[2] SESSIONS` 的 `[Alt-t]erm` | `TestTerminalPopupSize`、`TestTerminalCursorBlock` |
| terminal §按鍵 | 出口鍵可以設定（預設 `Alt-Esc`，`Alt-Esc` 一律先問、其他由 app 決定要不要問；lib：`Terminal.ExitKey`、`ExitAsks`）；保留鍵執行中能換；hint 與 footer 照當下的設定 | 測試驗 | `TestTerminalExitKey` |
| terminal §歷史模式 | `PgUp` 進入 History 模式（框與模式名 Yellow、`┫History┣`）；`j\|k`、`u\|d`、`gg\|G`、`PgUp\|PgDn` 捲歷史、不送給子程序；`Esc` 回即時畫面；其他鍵（含 `q`、`Ctrl-C`）先離開模式再送給子程序；hint／footer 與 `?` 照 K11 揭露 | T `[1] SHELL` 跑幾行後按 `PgUp` | `TestTerminalHistoryMode` |
| terminal §選取模式 | app 保留的組合鍵進入（展示場 `Alt-v`，常駐揭露；lib：`Terminal.SelectKey`）；畫面凍住、子程序後來的輸出先不畫，游標可以往上走進歷史；按鍵、顏色、捲動位置、複製後的 Success toast 照 note 的選取模式；離開回到即時畫面，之後按鍵照常送給子程序 | T `[1] SHELL` 按 `Alt-v`、`v`、移動、`y` | `TestTerminalSelectMode` |
| terminal §按鍵 | 按鍵都給子程序；hint 出口鍵第一、最後才捨；`Alt-Esc:exit`（結束）／`Alt-Esc:leave`（留著），不寫 `close`；後面跟保留鍵；`Alt-Esc` 先 confirm | 同上 | `TestK10ExitKeyShown`、`TestK10AltEscConfirms`、`TestEncodeKey` |

## dialog/toast

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| toast §長相 | popup 的框、一律第 1 層層色；不觸發淡化、也不被淡化 | D「Toast over a popup」 | `TestToastNotALayer` |
| toast §長相 | 畫面下方水平置中；下框一律在畫面倒數第三列，最下面兩列一直看得到（平常是 panel 下框與 footer；terminal 開著時是 PTY 最後一列與 terminal 下框）；寬度照「內容確定」 | P 的 `Toast info`；T 的 terminal popup 裡按 `Alt-b`（卡住結束時跳 toast） | `TestToastPosition`、`TestToastPositionOverTerminal` |
| toast §長相 | 文字置中、太長折行每列置中；最多幾列由 app 設（lib：`App.ToastLines`，預設不限），超過時最後一列結尾 `…`；不高過畫面倒數第三列以上放得下的 | `Toast long` | `TestToastWrapLines` |
| toast §長相 | Success U+F05E0 `SUCCESS` Green 字；Info U+F024 `INFO` Text；Warning U+F078F `WARNING` Peach；Error U+F0238 `ERROR` Red | P 的 `Toast success`、`Toast info`、`Toast warning`、`Toast error` | `TestToastKinds` |
| toast §長相 | Success、Info 2200ms，Warning、Error 4400ms；第二個來直接取代、重新計時 | 連按兩次 `Toast …` | `TestToastDuration`、`TestToastReplace` |
| toast §按鍵 | `Esc` 立刻收、比其他 popup 先收；其他鍵穿過；PTY 裡只能等時間到；hint `Esc:close` | toast 開著時按 `j`；T 的 terminal popup 裡按 `Alt-b` | `TestToastEscFirst`、`TestToastKeysPassThrough` |

## input/README（所有 input 共用）

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| input §什麼是 input | 打字列上是輸入態；不打字的部分 core key 照 K1（`?` key reference、`q`／`Ctrl-C` 離開、`Space` 不作用、`hjkl` 移動）；`Enter` 照 K3；`Esc` 關值不變；從表單開的確定後 focus 留原欄 | I `[1] INPUTS` 每一列 | `TestInputCommon` |
| input §什麼是 input | 錯誤點名的是值的名字：從表單欄位開的是那一欄的名字；自己打開、標題寫在做什麼（`RENAME`、`SAVE AS`）的，用 app 另外給的名字（`Name`；沒給時用標題，是 lib 的退路）；text、password、number、textarea 都一樣 | I 的 `Name`（標題 `RENAME`）貼上多行按 `Enter`：`Name can't have control characters`；`ErrEmpty` 一樣點名（`Name can't be empty`） | `TestInputErrorNamesValue`、`TestPasteBlockedOnEnter`、`TestInputErrEmptyNamesValue` |
| input §什麼是 input | 有按鈕的 input popup 照表單的動作區：`Tab\|Shift-Tab` 在欄位區與動作區之間來回、回來停在離開前那一項；hint 在欄位上 `Tab:actions`、在按鈕上 `Tab:fields`。lib 裡有按鈕的 input popup 只有 color picker | I 的 `Background`（color） | `TestColorPickerTab`、`TestColorPickerKeys` |
| input §單行骨架 | 說明列一句話 Overlay0、空一列；打字列最前面 `nf-fa-angle_right`（層色，離框一格）、空一格接值；值 Lavender、游標 Lavender 底 Base 字；錯誤列；hint `Enter:<動詞> Esc:cancel`（列出全部操作）；寬度：自由打字用上限、有長度上限照內容 | I 的 `Name`、`Timeout`、`Password` | `TestSingleLineSkeleton`、`TestInputAngle` |
| input §值比框長時 | 顯示游標附近那段、截掉的那邊補 `…`；游標一定看得到、只捲剛好的量；照顯示寬度、一個字不切半（紅色 `\n` 也是） | I 的 `Name`（打一串長的），`Home`／`End` | `TestInputLongValue` |
| input §編輯鍵 | 字元、`Space` 插入；`←\|→`；`Home\|End`；`Backspace`、`Delete`；一個字是看起來的一個字；不收 `Ctrl-U`、`Ctrl-W`；其他 `Ctrl-`、`Alt-` 不作用不打字 | I 的 `Name`（打中文、emoji、組合字元） | `TestInputEditKeys`、`TestInputGraphemes`、`TestInputCtrlAltIgnored` |
| input §灰字提議 | 只用在單行（text、number）；打開是空的、舊值灰字從游標那一格開始（游標壓在第一個字上，Lavender 底、Base 字）；`Tab` 收進來；直接打字換新值（刪光再出現）；空的時候 `Backspace` 丟掉舊值、`Enter` 存空；什麼都沒碰 `Enter` 不改；hint 加 `Tab:edit Backspace:clear`；兩種來源：舊值與沒有舊值時 app 給的建議值，不同時出現 | I 的 `Name` | `TestInputSuggestion`、`TestSuggestionUnderCursor`、`TestInputGreyTakesABurst`（一次送進一整串字照樣打進去） |
| input §新建一個值：建議值 | 沒有舊值、app 有合理的預設時用灰字提議（text、number）；長相、`Tab`、打字、`Backspace`、hint 跟舊值一樣；什麼都沒碰按 `Enter` 用建議值、照常檢查、寫回；不能空又有建議值時 `Enter` 用建議值，`Backspace` 丟掉後再 `Enter` 才是「不能空」（lib：`TextInput.Suggest`、`NumberInput.Suggest`） | I 的 `Archive`（`assets.zip`） | `TestInputSuggestedValue` |
| input §不放 placeholder | input popup 裡不放 placeholder | I 每一列 | `TestInputNoPlaceholder` |
| input §貼上的控制字元 | 一行的值：貼上的控制字元一個都不丟、畫成紅字不切開：`\r` `\n` `\t` 寫名字（2 格）、其他 C0、DEL、C1 寫 `\xNN`（4 格）；`\r\n` 是兩個；灰列裡跟著灰；游標停在上面時整個控制字元 Red 底、Base 字（textarea 的寫與移也一樣）；`Backspace` 整個刪；遮罩照遮 | I 的 `Name` 貼上多行、貼上帶顏色的字，`←` 移到控制字元上 | `TestPasteMarkers`、`TestPasteControlChars`、`TestCursorOnControlChar` |
| input §貼上的控制字元 | 不是打字來的值走同一個過濾；會被拿去用的在 `Enter` 時擋、錯誤列點名值（`Name can't have control characters`）；在去空白字元之前檢查；搜尋與篩選的打字列只畫不擋 | I 的 `Name`；I `[/] Find` 貼上 | `TestPasteBlockedOnEnter`、`TestPasteSearchNotBlocked`、`TestSearchInputNotBlocked`（只拿來搜尋的 text input：`TextInput.Search`） |
| input §值 | 一行的值確定時去掉前後的空白字元（所有 Unicode 空白：一般空格、U+3000、NBSP…）；password 不去 | I 的 `Name` 打 `　x ` | `TestInputTrimWhitespace` |
| input §在表單與 panel 上怎麼顯示 | 控制字元照實寫成紅色的轉義（`\n`、`\x1b`、`\t`），不丟、不換成空白；app 自己的顏色照留 | I 的 `Separator`（`\t`） | `TestValueControlChars` |
| input §在表單與 panel 上怎麼顯示 | Text；空值灰字寫「空著代表什麼」（app 寫、不教按鍵）；太長尾端截、路徑前面截；各 input 的顯示（password 8 個 `•`、textarea 第一行 + `…`、checkbox popup 用 `, ` 接、reorder 照目前的順序用 `, ` 接、color 色塊 + hex、file `~`…） | I `[1] INPUTS` 每一列（含空的）；家目錄寫成 `~`：`Download to` | `TestValueDisplay` |

## input — 各種

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| text | 單行骨架、編輯鍵、提議、貼上照 README；確定去空白字元 | I 的 `Name`、`Path` | `TestTextInput` |
| password §長相 | 兩個方案：一般（每字一個 `•` 靠左）、解鎖（每字一顆 `●` 間隔一格、置中往兩側長）；遮罩 Lavender；舊密碼 8 個灰點用同方案的字；放不下照「值比框長時」；有長度上限照內容寬 | I 的 `Password`（一般）、`PIN`（解鎖，12 碼，框 43 格） | `TestPasswordMasks`、`TestPasswordUnlockWidth` |
| password §解鎖的游標 | 游標在下一顆 `●` 會出現的位置；打滿最後一碼時停在最後一顆上（Lavender 底、Base 的 `●`）；置中連游標一起算（n 碼 2n + 1 格、打滿 2N − 1 格）；解鎖沒有 `nf-fa-angle_right` | I 的 `PIN` 打幾碼、打滿 12 碼 | `TestUnlockCursor` |
| password §長相 | 解鎖沒有說明列，要輸入什麼由標題講；框裡兩列空白、打字列、一列空白、錯誤列，連框 7 列，打字列在正中間 | I 的 `PIN`（`NEW PIN`）、`Unlock` | `TestPasswordUnlockRows` |
| password §狀態 | 打開是空的、有舊密碼畫 8 個灰點，游標壓在第一顆上（解鎖樣式 8 顆間隔一格、置中、15 格）；什麼都沒打 `Enter`／`Esc` 保留；打字換新的；沒有 `Tab` | I 的 `Password`（已設）、`PIN`（已設） | `TestPasswordExisting`、`TestOldPasswordDotsCursor` |
| password §按鍵 | 貼上照 README（照遮）；`Backspace` 一次清全部（打的 → 回到灰點；空的 → 清掉舊密碼、`Enter` 存成沒有）；hint 加 `Backspace:clear`；只接在尾端，`←→` `Home` `End` `Delete` 沒作用 | 同上 | `TestPasswordBackspaceClears`、`TestPasswordAppendOnly` |
| password §值 | 不去空白字元；表單與 panel 上設了是 8 個 `•`（Text）、沒設照空值 | I `[1] INPUTS` 的兩列 | `TestPasswordDisplay` |
| number | 說明列寫範圍；範圍已知寬度確定；打字只收可能的字（負數 `-`、小數 `.`）、貼上只留會收的；不做 `↑↓` 加減；範圍在 `Enter` 檢查 | I 的 `Port`（1-65535）、`Ratio`（小數、可負） | `TestNumberFilter`、`TestNumberRange` |
| number §值 | 欄位可以空時空的值不做範圍檢查、存成空的，空著代表什麼由 app 用灰字寫；不能空時空的值照錯誤處理（lib：`NumberInput.Optional`；不能空的錯誤是 `ErrEmpty`：`Port can't be empty`） | I 的 `Refresh`（`30 seconds (default)`） | `TestNumberOptional`、`TestK3InputUnchangedNoWrite` |
| number §長相 | 單位：打字列下面一列單位列，完整的字、Overlay0、跟值的第一格對齊、不跟著打的字移動；寬度算進去；表單與 panel 上由 app 寫成 `10 seconds`（lib：`NumberInput.Unit`） | I 的 `Refresh` | `TestNumberUnit` |
| textarea §長相 | 太長自動折行（在空白處斷，見 components §折行）、不左右捲；行號每行第一列、靠右、寬度 max(2, 最大行號位數)、後空一格、折下來的空白；行號 Overlay0、游標那行 Text；寬度與高度都用上限、內容在框裡捲，放不下時捲動位置寫游標的行 `N of M`；值裡的 `\t` 畫成 `input_tab_width` 格 | I 的 `Comment` | `TestTextareaWrapAndNumbers`、`TestTextareaGutterTwoDigits` |
| textarea §按鍵 | 寫（輸入態、`Enter` 換行、`Tab` 插 `input_tab_width` 個空白（設定、預設 4）、hint `Esc:move`、游標 Lavender 一格）／移（游標層色一格；`Esc` 進來、`i` `a` `o` 回寫、`Enter` 確定、hint `Enter:save i:write Esc:cancel`）；移裡 `Esc` 關（改過先 confirm）；打開在「寫」、舊值帶入、游標在最後、沒有灰字提議 | 同上（有舊值） | `TestTextarea` |
| textarea §按鍵 | 寫的編輯鍵加 `↑↓`（照畫出的列）、`←→` 跨行、行首 `Backspace` 合併、`Home\|End` 到整篇的頭尾；移的鍵 `hjkl` `w\|b\|e` `0\|$` `u\|d` `gg\|G`；移不是模式也不是輸入態（`?` 開 key reference） | 同上 | `TestTextareaWriteKeys`、`TestTextareaMoveKeys`、`TestTextareaHomeEnd` |
| textarea §按鍵 | 貼上的換行（`\n`、`\r\n`、單獨的 `\r`）是真的換行；其他控制字元留在值裡畫紅字、`Backspace` 整個刪；有控制字元時「移」裡的 `Enter` 擋下，錯誤列點名值（`Notes can't have control characters`），換行與 `\t` 不算 | I 的 `Comment` 貼上帶顏色的字 | `TestTextareaPasteControlChars` |
| finder §長相 | 兩塊：打字列（最前面 U+F105、離框一格、空一格接打的字；focus 在打字列時層色、在清單時 Overlay0；右邊 `符合/總數`，Overlay0）、分隔線、清單；清單的 cursor 從框貼到框 | I `[/] Find` | `TestFinderLayout` |
| finder §長相 | 一般 finder 一定有預覽（select、checkbox popup 沒有）；兩框合起來一個 popup 寬；高度一律取最高；寬 ≥ 96 左右並排各一半（有 metadata 時右框是預覽／分隔線／四列 metadata）；不到 96、有 metadata：上面是清單框與 `INFO` 框各一半、下面是整條寬的預覽框，上下各一半；不到 96、沒有 metadata：清單在上、預覽在下，各一半；預覽標題是那一筆的名字、不拿 focus；預覽檔案或目錄的 finder 照 file-picker 的預覽，其他的內容 app 給、沒有 metadata；預覽框用層色、內容不淡化、下框不寫字 | 有 metadata：I `[2] FILES` 的 `[/] Find`；沒有 metadata、預覽是 app 給的：I 的 `Host`；寬 120 與 80 各看一次 | `TestFinderPreview`、`TestFinderFileLayout` |
| finder §長相 | 邊找邊列：標題後 loading icon、一筆都沒到寫 `(indexing…)`；高度打開時定、寬度用上限 | I `[2] FILES` 的 `[S]low index` | `TestFinderStreaming` |
| finder §清單的標記 | select 前面 radio glyph、checkbox popup 前面 checkbox glyph；目前的值字 Green；cursor 在目前的值上照 cursor | I 的 `Profile`、`Namespaces` | `TestFinderMarkers` |
| finder §狀態 | 只有 focus 那塊亮：focus 在打字列時清單 cursor 是層色 cursor 再淡化；focus 在清單時打字列整列灰、沒游標 | I `[/] Find` 按 `Tab` 來回 | `TestFinderFocusBrightness` |
| finder §按鍵 | 打字列：字元篩選、`↑↓` 移 cursor（繞回）、`Enter` 到 cursor 那項（沒結果不作用）、`Tab` 到清單、`Esc` 關整個（filter 是清掉）；清單：移動（繞回）、`Enter` 去那裡（沒有結果不作用，checkbox popup 例外）、`Tab\|/` 回打字列接著舊的字打、`Esc` 關整個 | 同上 | `TestFinderTypingKeys`、`TestFinderListKeys` |
| finder §比對 | 比對的對象由 app 決定：預設每一項顯示的字，也可以是其他內容（lib：`FinderItem.Match`）；搜尋要花時間時（讀每個檔、交給搜尋工具或遠端一次找一批）邊找邊列、打字列照常能打（lib：`Finder.Search`，app 自己找、結果放進 `Items`，lib 不再篩） | I `[2] FILES` 的 `[F]ind in files` | `TestFinderMatchesWhatTheAppSays` |
| finder §比對 | 包含、不分大小寫、字要連續出現；空白分開的每個詞都要出現、順序不拘；結果保持清單本來的順序（數字照數值、文字照文字）；用在 finder、select、checkbox popup、file-picker 的一般字、panel filter | I `[/] Find` 打 `read`、`READ me`；P filter 打 `run 3d` | `TestMatchContains`、`TestMatchWords`、`TestMatchKeepsOrder` |
| finder §hint | 打字列 `Enter\|Tab:list Esc:cancel`；清單 `Enter:<動詞> Tab\|/:filter Esc:cancel` | 同上 | `TestFinderHint` |
| finder §hint | 沒有結果時（邊找邊列、一筆都還沒到也是）兩處的 `Enter` 都不作用，hint 拿掉 `Enter`：打字列 `Tab:list Esc:cancel`、清單 `Tab\|/:filter Esc:cancel`；有結果時放回來；select、file-picker 也一樣（G93） | I `[/] Find` 打 `zz`；I 的 `Profile`（select）篩到沒有 | `TestFinderHintNoResults` |
| select | finder 骨架沒預覽、打字列一直顯示；radio glyph、目前的值 Green；選項可有熱鍵（app）；寬度照選項；打開 focus 在清單、cursor 在目前的值；`Enter` 選定寫回關掉（從表單開的 focus 留原欄）；hint 清單 `Enter:choose Tab\|/:filter Esc:cancel`、打字列 `Enter\|Tab:list Esc:cancel` | I 的 `Profile`（帶數字熱鍵） | `TestSelect` |
| radio | 五個以內畫在表單或 panel；U+F043E／U+F043D；選中 Green 其他 Text；一律直排、一個選項一列；每個選項一個項目；`Enter` 選、focus 不動、馬上生效 | I 的 `Layout`、`Order` | `TestRadio` |
| checkbox §長相 | U+F0132（勾）／U+F0131；勾的字 Green；popup 是 finder 骨架沒預覽、checkbox glyph、寬度照選項 | I 的 `Notify`（開關）、`Agent`（一組）、`Namespaces`（popup） | `TestCheckbox` |
| checkbox §按鍵 | 表單或 panel 上 `Enter` 原地翻；popup：打開在清單、`Space` 只改畫面、`Enter` 整組寫回才生效（從表單：寫回並標改過；單獨：這時才套用；沒改就關）、`Esc` 取消不問；勾的字跟著畫面上的勾變 Green；hint `Enter:apply Space:toggle Tab\|/:filter Esc:cancel`；篩到沒有結果時清單上的 `Enter` 照樣寫回整組（被篩掉的勾也算），hint 拿掉 `Space`：`Enter:apply Tab\|/:filter Esc:cancel`，打字列照 finder `Tab:list Esc:cancel` | I 的 `Namespaces`；D「Form: New host」的 `Namespaces` | `TestCheckboxPopup` |
| checkbox §按鍵 | 打開時 cursor 停在 app 指定的那一項，沒指定是第一項（ccdrawu 的 CD8）。lib：`Checks.Cursor`，捲到看得見 | 展示場都從第一列開（預設） | `TestCheckboxPopupCursor` |
| checkbox §長相、§按鍵 | 不可用的選項（not available）照樣畫在原位，整列 Surface2、勾也是；cursor 停在上面照 cursor 色塊畫；操作跟其他選項一樣（cursor 照樣停、`Space`／`Enter` 照樣翻、popup 的 `Enter` 照畫面上的勾寫回）；篩選照樣列出；代表什麼由 app 決定。不是停用：表單的停用選項照舊跳過。lib：`Checks.Unavailable`、`FormField.OptionUnavailable`、`Field.OptionUnavailable` | I 的 `Namespaces`：`kube-system`（勾著）與 `vault`，打 `vau`；I 的 `Agent` 與 D「Form: New host」的 `Agent`：`X11` | `TestCheckboxPopupUnavailable`、`TestCheckboxUnavailableInPlace` |
| slider | 表單與 panel 上軌道 `━` + `●` + 數字（寬度與顏色 app 定）；`Enter` 開 select（bottom 到 top、每 step 一個，cursor 在目前值，打數字篩選）；沒有自己的鍵；bottom、top、step 必有，step 預設 `(top − bottom) / 10`、可以是小數 | I 的 `Brightness`（0–100）、`Opacity`（0–1，step 0.05） | `TestSlider` |
| datetime §日期 | 月、年各是可 focus 的項目（focus 時照 cursor）、之間兩格；日曆靠左離框一格、不另外縮；每天 4 格（日期 2 格靠右 + 間距 2）；固定 6 列；cursor 那天就是要寫回的那天；原本的值 Green、打開 cursor 在它（沒有就今天）；今天底線；週末不上色；一週從週一或週日（app）；範圍外 Surface2、cursor 不停 | I 的 `Due date`（有範圍、週一開始） | `TestDatePickerLayout` |
| datetime §日期加時間 | 右邊直分隔線（不接框、左右各空兩格）；時 00–23、分每 5 分、兩欄之間兩格；目前的值 Green；跟日曆一樣高、超出捲；值不在格上停在之前最近的；沒 focus 那塊與時間塊裡沒 focus 那一欄的 cursor 再淡化（整個 picker 只有一個亮的 cursor）；版面固定、寬度確定 | I 的 `Meeting`（值 10:07） | `TestDatetimePickerLayout`、`TestDatetimeOneBrightCursor` |
| datetime §按鍵 | `Tab`：月 → 年 → 日曆 → 時間 → 月，打開在日曆；月年 `h\|l` 換、日曆跟翻（沒那天停最後一天）；日曆 `h\|l` 一天 `j\|k` 一週、跨月、跳過範圍外、不繞；時間 `j\|k` 欄內（繞回，各欄自己繞）、`h\|l` 換欄；`Enter` 各塊；`t` 今天（時間到現在、往前取 5 分）；`Esc` 不改；hint 五種 | 同上 | `TestDatetimePicker`、`TestDatetimeTimeWraps`、`TestDatetimeEnter`、`TestDatetimeHint` |
| datetime §值 | 可設最早最晚；表單與 panel 上格式 app 定，空照空值 | I `[1] INPUTS` | `TestDatePickerRange` |
| color-picker §長相 | 一列：old、new、R、G、B，五個項目之間兩格；old／new 色塊 8×2 + 名字 + hex；R、G、B 色塊 3×1（畫該色版自己的顏色）+ hex；下面分隔線與 `Choose` 按鈕（照表單的按鈕）、沒有錯誤列；寬度確定，照最寬的 hint 是 40 格 | I 的 `Background` | `TestColorPickerLayout` |
| color-picker §按鍵 | 可 focus：R、G、B、`Choose`；hjkl 只在 R、G、B 之間（`h\|l`、`j\|k` 前後、到頭尾繞回），`Choose` 只用 `Tab` 到；old、new 不拿 focus；打開在 R；focus 的色版下面兩列（名字與 hex）照 cursor、跟色塊一樣寬；R/G/B `Enter` 開 select 00–FF（`2A (42)`、hex 十進位都能篩），回來 new 跟著變、focus 留著；`Choose` 上 `Enter` 確定；`Esc` 不改；`Tab\|Shift-Tab` 在色版與 `Choose` 之間來回、回來停在離開前的色版；hint 兩種（`Enter:edit Tab:actions Esc:cancel`、`Enter:choose Tab:fields Esc:cancel`） | 同上 | `TestColorPickerKeys`、`TestColorPickerTab`、`TestColorPickerChannelSelect` |
| color-picker §長相 | 原本沒有值時 old 不畫色塊、寫 `none`（Overlay0），new 從 `#000000` 起；按 `Choose` 一律算改了，`#000000` 也寫回（lib：`ColorPicker.Value` 沒有 `Set`） | I 的 `Highlight` | `TestColorPickerNoColour` |
| color-picker §值 | hex 一律大寫；表單與 panel 上 3 格的色塊、空一格、hex；cursor 在這一欄時色塊保留自己的顏色、hex 換成 Base | I `[1] INPUTS` 的 `Background` | `TestValueDisplay`、`TestSwatchOnCursor` |
| file-picker §長相 | 根目錄 app 給、只是起點，最上面 `..`（`/` 沒有）；目前的目錄寫在標題（欄位名後，太長從前截）；打字列打開是空的；每列前目錄／檔案 icon（顏色 app 定）；原本的值 cursor 停在它、字 Green；`..` → 目錄 → 檔案、照名字不分大小寫，隱藏檔只在篩選字 `.` 開頭時列（`*` 不比對隱藏檔、`**` 不走進隱藏目錄）；打字列前面 U+F105；四塊：打字列、清單、預覽、metadata；預覽最多讀 64 KB：目錄與壓縮檔是三層的樹（`├─` `└─` `│` Overlay0、隱藏檔不列、滿了最後一列 `…`、`(empty)`、`(permission denied)`）、圖片第一列灰字 `data:image/png · 12.4 KB` 空一列接 data URI（超過 1 MB 只寫一句）、PDF 第一列灰字頁數空一列接前 20 頁的文字（最多 128 KB，沒有文字寫 `(no extractable text)`）、文字檔上色加行號、其他是十六進位一律一列 8 bytes 接 ASCII（33 格，並排、上下疊都一樣）、讀不到 `(unreadable)`；metadata 四列 `TYPE`（`text` `private key` `image` `PDF` `archive` `binary`、`link → <目標>`、`directory`）`SIZE`（1024 進位一位小數、目錄寫 `N items`、一項寫 `1 item`，不算隱藏檔）`MODIFIED`（本地時間到分）`MODE`（`ls -l` 的寫法），symlink 三列寫目標的，拿不到的寫 `—`（讀不到的目錄的 `SIZE`、斷掉的 link 的三列、列出之後才不見的檔的每一列），label 層色、值 Text；設定 `hide_secrets_in_preview`（預設關）打開時私鑰露第一行接灰字、其他只有擁有者能讀的檔一行都不露（目錄不算），metadata 照常 | I 的 `Identity file`（假的 `.ssh`，打 `/tmp/terminu-showcase/project/` 走到假的專案：`empty` 是 `(empty)`、`locked` 是 `(permission denied)`）；I `[2] FILES` 的 `[/] Find`（假的專案，每一種檔各一個：`latest.pdf` 是 link、`broken.pdf` 是斷掉的 link、`docs/scan.pdf` 沒有文字、`assets/photo-large.png` 超過 1 MB、`secret.bin` 讀不到）；global 的 `Hide secrets in preview` | `TestFilePickerLayout`、`TestFilePickerHiddenAndOrder`、`TestFilePreviewKinds`、`TestFilePreviewEscapes`（檔案裡寫的 escape 序列照字畫、ESC 寫紅色的名字、一個字都不丟）、`TestLsMode`（`MODE` 照 `ls -l`：`drwxrwxrwt`、`-rwsr-xr-x`）、`TestFilePreviewHidesSecrets`、`TestFileInfo`、`TestFilePickerMetaDash`、`TestFinderFilesInFS`（預覽讀 app 給的檔案系統：`PreviewFS`、`Finder.FS`）、`TestFilePreviewPanel`（從某一行開始：`FinderItem.Line`，那一行在第三列、上面兩行前文、不另外標、行號照檔案真實的行號） |
| file-picker §按鍵 | 清單 `Enter`：目錄進去（清篩選、cursor 在 `..` 下第一項）、檔案選中寫回關掉；選檔時目錄永遠不會被選中 | 同上 | `TestFilePicker` |
| file-picker §選目錄 | 清單只列目錄，`.` → `..` → 目錄；最上面 `. (this directory)`、icon U+F0252；在 `.` 上 `Enter` 選目前的目錄（`Enter:choose`），在目錄上 `Enter` 走進去（`Enter:into`）；打字列有篩選字或 wildcard 時沒有 `.`、wildcard 結果只列目錄；其他照 file-picker | I 的 `Path`、`Download to`（**`Download to` 打開的是真的 `~/Downloads`**） | `TestFilePickerDirMode` |
| file-picker §按鍵 | 打字列：一般字照「比對」篩選目前這一層；路徑（`/`、`~/` 開頭）一直留在打字列、以最後一個 `/` 切成目錄與篩選字；目錄不存在或有 `..` 這一段：清單是空的；`*` `?` wildcard（`**` 往下找）、每列寫相對於起點的路徑、標題寫起點、沒有 `..`；hint 三種 | 同上，在打字列打 `/tmp/terminu-showcase/.ss`、`/tmp/terminu-showcase/.ssh/id_`、`/tmp/terminu-showcase/nope/x`、`/tmp/terminu-showcase/../x`、`/tmp/terminu-showcase/.ssh/**/*.pem`（`~/` 開頭的路徑是真的家目錄） | `TestFilePicker`、`TestFilePickerPathRules`、`TestGlobMatch`、`TestFilePickerHint` |
| reorder §用途 | 一項一項搬、排出順序，值是整個順序；就是 panel 的 Drag mode 放進 popup；跟排序選擇器不同；沒有搜尋，不是 finder | I 的 `Segments` | `TestReorderDrag` |
| reorder §長相 | 每一列列首一格給 glyph、空一格接項目（Text），這一格一直保留；cursor 照 menu（層色底、Base 粗體、箭頭）；拿著的一項 Yellow 底、Base 粗體、箭頭 Yellow、列首 U+F04E2，框與模式名 Yellow、上框右側 `┫Drag┣`；標題寫排的是什麼；寬度打開時確定：最寬的項目 + 列首兩格 + 4，至少 35 格（hint 29 + 6）；放不下時捲、`N of M` | I 的 `Segments`（`D` 拿起） | `TestReorderLayout`、`TestReorderDrag`、`TestReorderScroll` |
| reorder §按鍵 | 打開在清單；沒拿東西：`j\|k` 移 cursor 繞回、`u\|d` 半頁、`gg\|G` 頭尾；`D` 拿起；`Enter` 寫回整個順序（沒變不寫）；`Esc` 沒改過直接關、改過先問 `Discard changes to <標題>?`（`Esc` 回清單、順序還在）；「改過了」放下的位置不同就設、不清掉；`Space` 不作用；hint `Enter:apply D:drag Esc:cancel` | I 的 `Segments` | `TestReorderDrag`、`TestReorderEsc`、`TestReorderSpace` |
| reorder §按鍵 | 拿著一項（Drag mode）：`j\|k` 跟下一項、上一項交換，cursor 跟著，到頭尾繞回；`Enter` 放下、cursor 留在那一項；`Esc` 放回拿起前的位置；其他鍵不作用（`D` 不是放下）；hint `Enter:drop Esc:cancel`；`?` 是 Drag mode 的鍵 | 同上 | `TestReorderDrag`、`TestReorderDragWraps`、`TestReorderDragModeKeys` |
| reorder §值 | 表單與 panel 上照目前的順序用 `, ` 接，太長加 `…` | I `[1] INPUTS` 的 `Segments` | `TestValueDisplay` |

## 熱鍵標記（M5）在展示場用到的地方

| 情境 | 展示場的登記 |
|---|---|
| 首字母 | `[m]ark`、`[D]rag`、`[V]iew` |
| 首字母，大小寫跟 label 不同 | `[v]im`、`[e]xport`（D「Menu: Open with」、「Menu: long descriptions」） |
| 在 label 中間 | `Copy UR[L]`（I `[2] FILES`）、`He[x] view`（D「Menu: Open with」） |
| 字母不在 label 裡 | `[x] Delete` |
| 含 modifier | `[Alt-t]erm` |
| 多字元 | `[go]to` |
| 數字 | select 選項的 `[1] clock`、`[2] matrix`；global operation popup 的 `[1] About` |
| core key | `[Enter] Logs` |
| 沒有熱鍵 | `Empty`、`Loading`、`Fail` |
| label 本身寫了鍵 | `[/] Find`、`[/] Filter` |

## glyph（C4 的清單）

規範用 code point 指定的 glyph。C4 要每一個在展示場出現一次、截圖看三件事（游標前進幾格、清不清楚、意思對不對得上）。

| code point | 名稱 | 規範 | 展示場 |
|---|---|---|---|
| U+E0BA | `nf-ple-lower_right_triangle` | panel §膠囊串（左端） | 每個 panel 標題、chip 列、popup 的 tab |
| U+E0B0 | `nf-pl-left_hard_divider` | panel §膠囊串（右端、換色接縫）；popup §外框（cursor 尾端的箭頭） | 同上；每個 cursor |
| U+E0B1 | `nf-pl-left_soft_divider` | panel §膠囊串（同色接縫） | 畫面 chip 列（沒選中的 chip 之間） |
| U+E0B6 | `nf-ple-left_half_circle_thick` | popup §外框（標題膠囊左端） | 每種 popup |
| U+E0B4 | `nf-ple-right_half_circle_thick` | popup §外框（標題膠囊右端） | 每種 popup |
| U+F02D8 | `nf-md-hexagon` | menu §長相、note §key reference（區塊標題） | Space menu；任一處按 `?` |
| U+F160 | `nf-fa-sort_amount_asc` | panel §一般清單（升冪；C3） | P `[1] PODS` 表頭 |
| U+F161 | `nf-fa-sort_amount_desc` | panel §一般清單（降冪；C3） | 同上 |
| U+F105 | `nf-fa-angle_right` | finder §長相、panel §panel filter、file-picker（打字列最前面） | I `[/] Find`；P `[1] PODS` 按 `/` |
| U+F0252 | `nf-md-folder_move` | file-picker §選目錄（`. (this directory)`） | I 的 `Path`、`Download to` |
| U+F0132 | `nf-md-checkbox_marked` | panel §一般清單（標記）、checkbox、finder §清單的標記 | P 標記的列；I 的 `Agent`、`Namespaces` |
| U+F0131 | `nf-md-checkbox_blank_outline` | checkbox、finder §清單的標記 | I 的 `Agent`、`Namespaces` |
| U+F043E | `nf-md-radiobox_marked` | radio、select | I 的 `Layout`、`Profile` |
| U+F043D | `nf-md-radiobox_blank` | radio、select | 同上 |
| U+F0C9 | `nf-fa-bars` | popup §外框（menu） | Space menu |
| U+F059 | `nf-fa-question_circle` | popup §外框、note §key reference | 任一處按 `?` |
| U+F0ECC | `nf-md-shield_alert` | popup §外框、confirm | D「Confirm」 |
| U+F04E2 | `nf-md-swap_vertical` | panel §一般清單（Drag 拿著的列）、reorder §長相 | P `[1] PODS` 的 `[D]rag`；I 的 `Segments` 按 `D` |
| U+F018D | `nf-md-console` | popup §外框、terminal | T 的 terminal popup |
| U+F05E0 | `nf-md-check_circle` | toast（Success） | P 的 `Toast success` |
| U+F024 | `nf-fa-flag` | toast（Info） | P 的 `Toast info` |
| U+F078F | `nf-md-alarm_light` | toast（Warning） | P 的 `Toast warning` |
| U+F0238 | `nf-md-fire` | toast（Error）、note §error popup | P 的 `Toast error`；D「Error popup」 |
| U+F0A9E–U+F0AA5 | `nf-md-circle_slice_1` – `_8` | popup §loading | D「Loading popup」；P `Loading` |
| U+F0C8 | `nf-fa-square` | rules S5（mark 的每一格） | P `[1] PODS` 按 `V` |

規範直接寫出字元（不是 PUA）的符號，一起看：`┏━┓`（focus 的 panel、popup）、`┌─┐`（失焦的 panel）、`┫ ┣` `┤ ├`（模式名的接頭）、
`━`（畫面 chip 列下的分隔線、開關動畫的線、slider 軌道）、`╌`（U+254C，menu 區塊之間）、`─`（popup 裡的分隔線）、`│`（datetime 的
直分隔線）、`…`（截斷）、`•`（password 一般遮罩、表單上的 8 個點）、`●`（password 解鎖遮罩、slider 的值）、`███`（color 的色塊）、
紅色的 `\n` `\t`（貼上的換行與 Tab）。
