# tdp v0.5.3 coverage

tdp v0.5.3 逐條整理成清單：**規範出處 → 展示場的哪裡、怎麼按到 → 測試名稱**。這份清單就是展示場「所有的組合」的定義
（[`CLAUDE.md`](../CLAUDE.md) 的「展示場」）。看不到畫面的規定寫怎麼驗。

- 「展示場」欄是位置，「測試」欄是測試名稱；還沒寫出來的是預定的。寫好一條就把它變成真的；位置或名稱改了就改這裡。
- 出處寫法：`K4` 是 `principle/rules-zh_TW.md` 的規則；`P3` 是 `principle/README-zh_TW.md`；`popup §外框` 是
  `components/layout/popup-zh_TW.md` 的那一節，其他元件照檔名（`form`、`finder`…）；`components §大寫` 是 `components/README-zh_TW.md`。
- `G<n>` 是規範盲點紀錄裡還沒定案的編號（`~/Documents/sideproj/terminu/.local/family-fix/terminu-go/README.md`），定案前照紀錄的
  「暫時」做。G1–G22 已經補進 tdp v0.4.0、G23–G29 補進 v0.5.1、G30–G46 補進 v0.5.2（G43 不改規範）、G47 補進 v0.5.3，這裡直接寫規範的條文。
- 「lib」表示由 lib 的登記或 API 保證，不需要展示場的某一個畫面；測試照樣要有。

## 展示場

app 名 `showcase`，四個畫面，畫面 chip 列 `[P]ANELS [I]NPUTS [D]IALOGS [T]ERMINAL`（多畫面的 app；切換是 global operation，
`P` `I` `D` `T` 是 global 熱鍵）。每個畫面的 panel 各自從 `[1]` 編號。資料是假的，但排得像一個真的 app。

| 畫面 | 一句話 | panel |
|---|---|---|
| `[P]ANELS` | panel 的每一種樣子與狀態 | `[1] PODS`：一般清單（tab `PODS` / `EVENTS`、表頭排序、標記、cursor、40 列、截斷、panel filter、`[D]rag` 模式）；`[2] LOG`：串流內容、沒有 cursor；`[3] SETTINGS`：一列一個值 |
| `[I]NPUTS` | 每一種 input | `[1] INPUTS`：一列一個值，每一種 input（含各方案）各一列，`Enter` 打開它的 popup；`[2] FILES`：清單，`[/] Find` 開一般的 finder（有預覽） |
| `[D]IALOGS` | 每一種 dialog 與疊法 | `[1] DIALOGS`：每一列是一個情境（某一種 dialog、某一種狀態、某一種疊法），`Enter` 開它 |
| `[T]ERMINAL` | PTY 的兩種放法 | `[1] SHELL`：PTY 放在 panel 裡；`[2] SESSIONS`：清單，`[Alt-t]erm` 開 terminal popup |

**展示場自己的動作**（切換狀態用，放在 Space menu 的 panel operation，順便測了 M 章）：`Empty`（清空這個 panel）、`Loading`
（進入 loading，再按一次結束）、`Fail`（這個 panel 出錯）、`Disable`（把某些列與動作設成停用）、`Toast …`（Info / Warning /
Error / 長訊息）、`Start task`（畫面 chip 列下的進度條）。global operation popup 另有 `Narrow layout`（把窄寬的門檻拉到 80，
不用縮終端機也看得到窄寬）。icon 佔兩格用 `SHOWCASE__ICON_WIDTH=2` 啟動（E5 的覆寫），statusbar 寫出現在用幾格。

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
| 術語 模式 | 一部分鍵換意思、`Esc` 離開；zoom 這類版面切換不是模式 | P `[1] PODS` 的 `[D]rag`；D「Note viewer」的 `v`（visual） | `TestK11Mode*` |

## Rules — K Core key

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| K1 | 七種 core key（`Tab` `Enter` `Esc` `Space` `?` `q` `1`–`9`）在每個 surface 意義不變；例外只有輸入態、PTY、模式、表單上的 `q` | 每個畫面、每種 popup | `TestK1CoreKeysEverySurface` |
| K1 | letter hotkey 不能佔用 core key；app 可另定自己的 core key，定了就在所有 surface 一樣 | lib：登記時拒絕；自訂 core key 的 API | `TestK1HotkeyCannotTakeCoreKey`、`TestK1CustomCoreKeyEverywhere` |
| K1 | 數字鍵：panel 上（沒有 popup 時）跳到目前畫面的第 N 個 panel；沒有那個號碼不作用；換畫面就是那個畫面的 panel | P 按 `1` `2` `3`、按 `4`（不作用）；換到 I 按 `2` | `TestK1DigitsJumpPanels`、`TestK1DigitsPerScreen` |
| K1 | app 不能在 panel 上登記數字熱鍵；popup 裡數字自由（select 選項的數字熱鍵）；數字的 global 熱鍵只在 global operation popup 裡按 | lib：登記時拒絕；I 的 `Profile`（`[1] clock`）；global popup 的數字熱鍵 | `TestK1DigitHotkeyRejectedOnPanel`、`TestK1DigitsFreeInPopups`、`TestM4DigitGlobalOnlyInPopup` |
| K2 | `Tab`：畫面上換 panel、最後一個回到第一個；不跨畫面、不離開 popup | P 按 `Tab` 走過三個 panel | `TestK2TabCyclesPanels`、`TestK2TabStaysInPopup` |
| K2 | 表單：欄位；錯誤列（有錯誤時）與動作列也各是一站；表單一定有 `Shift-Tab` | D「Form: New host」 | `TestFormTabStops` |
| K2 | 有好幾塊的 popup（finder、datetime picker）：在塊之間 | I `[2] FILES` 的 `[/] Find`；I 的 `Meeting`（datetime） | `TestFinderTabBlocks`、`TestDatetimePicker` |
| K2 | 打字列（finder、select、panel filter）：到清單 | 同上；P `[1] PODS` 按 `/` | `TestFinderTypingTab`、`TestPanelFilterTab` |
| K2 | 只有一塊的 input popup：有灰字提議時接受提議，沒有時不作用 | I 的 `Name`（有舊值）與 `Nickname`（空的） | `TestInputSuggestionTab`、`TestInputNoSuggestionTabNoop` |
| K2 | 多行文字寫入狀態：`Tab` 是縮排；PTY 裡屬於子程序 | I 的 `Comment`（textarea）；T `[1] SHELL` | `TestTextarea`、`TestK10KeysToChild` |
| K3 | `Enter` 做 app 登記的那個動作；同一種項目同一個動作；panel 沒有項目時對整個 panel | P `[1] PODS` `Enter` 開 log；P `[2] LOG` `Enter` 開可捲動的檢視 | `TestK3EnterRunsRegistered`、`TestK3EnterOnContentPanel` |
| K3 | `Enter` 不移動 focus（表單送出失敗跳到第一個問題欄位是寫明的例外） | D「Form: New host」；P `[3] SETTINGS` | `TestK3EnterKeepsFocus` |
| K3 | input popup：確定寫回並關；不合格不寫回、錯誤列、不關；值沒變不寫回直接關 | I 的 `Port`（打 `0`） | `TestK3InputConfirm`、`TestK3InputInvalidStays`、`TestK3InputUnchangedNoWrite` |
| K3 | 多行文字：寫入狀態 `Enter` 換行，離開寫入後才確定 | I 的 `Comment` | `TestTextarea` |
| K3 | 打字列的 `Enter` 把 focus 移到清單 cursor 那一項、不執行；沒有結果不作用 | I `[/] Find` | `TestFinderTypingEnter` |
| K3 | 其他部分做最直觀的事（月份 → 日曆、R → 選值清單） | I 的 `Meeting`、`Background`（color） | `TestDatetimeEnterMonth`、`TestColorPickerEnterChannel` |
| K3 | 表單的 `Enter` 對 focus 的項目；送出只有按鈕，檢查全部，不合格不送、focus 跳到第一個 | D「Form: New host」 | `TestFormSubmitChecks` |
| K3 | menu、confirm：`Enter` 執行 cursor 那一列 / 接受 | Space menu；D「Confirm」 | `TestMenuEnterRuns`、`TestConfirmEnterAccepts` |
| K4 | `Esc` 關最上層 popup（含 toast）；沒有 popup 時退出模式或往上一層（清 filter）；一次一層；永不離開 app | P：開 filter、進 Drag、疊兩層 popup 後連按 `Esc` | `TestK4EscOneLayer`、`TestK4EscNeverQuits` |
| K4 | 多層 popup 只關最上層，底下原樣 | D「Stack: Space menu → Global operation → Quit」 | `TestK4EscOnlyTop` |
| K5 | panel 上 `Space` 開 / 關 Space menu；`Esc` 也能關 | 任一 panel | `TestK5SpaceToggles` |
| K5 | 其他 popup 上 `Space` 不作用；唯一例外 checkbox popup 的清單 | D「Confirm」上按 `Space`；I 的 `Namespaces` | `TestK5SpaceNoopOnPopups`、`TestCheckboxPopupSpace` |
| K5 | popup 的操作揭露在下框 hint 與 `?`，不在 popup 上疊 Space menu | 每種 popup | `TestK5NoMenuOnPopup` |
| K6 | `?` 在任何 surface 都有回應；再按 `?` 或 `Esc` 關；開最前端 surface 的 key reference | 每個畫面、每種 popup、模式裡 | `TestK6HelpEverySurface` |
| K6 | panel：這個 panel 能按的鍵（含移動鍵）＋這裡有作用的 core key ＋ global 熱鍵；popup：只有這個 popup 的熱鍵、`Enter`、`Esc`、移動鍵 | P `[1] PODS` 按 `?`；Space menu 上按 `?` | `TestK6PanelKeys`、`TestK6PopupKeysOnly` |
| K7 | 別名在所有 surface 一樣有效 | lib：別名登記一次、全域生效 | `TestK7AliasEverywhere` |
| K8 | 輸入態：letter hotkey、`Space`、`?`、`q` 是字元；`Esc` 取消；`Tab` 照 K2；`Ctrl-C` 進離開流程；離開輸入面立刻恢復 | I 的 `Name` 打 `q ?`；P filter 打 `q` | `TestK8InputState`、`TestK8CtrlCInInput` |
| K8 | textarea 寫入狀態 `Tab` 插入 `input_tab_width` 個空白（E3 的設定，預設 4），不插 `\t` | I 的 `Comment` | `TestTextarea` |
| K9 | `q` 與 `Ctrl-C` 同一個離開流程；`q` 在輸入態是字元、在表單上不作用；`Ctrl-C` 在輸入態與表單上有效；PTY 裡兩個都屬於子程序 | 任一 panel；D「Form」上按 `q` | `TestK9QuitSameFlow`、`TestK9QNoopOnForm` |
| K9 | 離開流程由 app 決定（直接 / 先確認 / 選擇）；進行中再按 `Ctrl-C` 立刻離開；離開列在 global operation popup | 展示場：先 confirm（`Quit showcase?`） | `TestK9SecondCtrlCQuits`、`TestK9QuitInGlobalPopup` |
| K10 | PTY 裡按鍵都給子程序；core key 與熱鍵失效 | T `[1] SHELL` 裡按 `Space ? q Tab Esc 1` | `TestK10KeysToChild` |
| K10 | 至少一個出口鍵（`Alt-Esc`），focus 在 PTY 時常駐揭露 | T：footer `Alt-Esc:leave …`；terminal popup 下框 | `TestK10ExitKeyShown` |
| K10 | `Alt-Esc` 離開或結束時一律先 confirm（`Enter` 離開、`Esc` 回 PTY）；PTY 裡面的動作不用問 | T `[1] SHELL`、terminal popup | `TestK10AltEscConfirms` |
| K10 | 其他 Alt 出口鍵要不要 confirm、保留哪些鍵、出口後 focus 落點：app 決定（lib 提供選項） | T `[2] SESSIONS` 的 `[Alt-t]erm` 隱藏 terminal popup | `TestK10ReservedKeysShown` |
| K10 | 子程序還沒準備好時可以不轉送一般鍵，但 `Ctrl-C` 照轉、出口鍵照有效照揭露 | T `[1] SHELL` 的 `Slow start`（模擬連線中） | `TestK10NotReadyForwardsCtrlC` |
| K10 | `Alt-Esc` 與兩次 `Esc` 的 byte 一樣（實作參考：bubbletea v1.3.10 實測會黏）：C2 驗 | 見紀錄 C2 | — |
| K11 | 模式裡 `Space` 不作用；`?` 開模式的 key reference；`Esc` 離開、回到進入前的狀態；`q`、`Ctrl-C` 照 K9 | P `[1] PODS` 的 `[D]rag` | `TestK11ModeCoreKeys` |
| K11 | 模式裡只有模式自己的鍵與上表的 core key 作用：panel 熱鍵與 global 熱鍵都不作用，按了不出聲 | Drag 裡按 `m`（panel 熱鍵）、`I`（global 熱鍵） | `TestK11HotkeysSilentInMode` |
| K11 | 模式可以暫停 `Tab` 與數字鍵，按了要有回應（toast）；toast 在時第一個 `Esc` 先收 toast | Drag 裡按 `Tab`、`2` | `TestK11TabPausedToast`、`TestK11DigitsPausedToast` |
| K11 | 模式名在框的上框右側、外框換模式色、離開恢復；不只靠顏色 | Drag；D「Note viewer」的 `v` | `TestK11ModeLabel` |
| K12 | 不在打字的地方 `j k u d g G h l` 保留給移動，任何動作都不佔 | lib：登記時拒絕（panel 與 menu；popup 裡數字可以） | `TestK12HotkeyCannotTakeMovement`、`TestK12HotkeyCannotTakeMovementInMenu`、`TestK1DigitsFreeInPopups` |
| K12 | 有左右結構：`h\|l` 左右（切 tab、跨側、日曆前後一天） | P `[1] PODS` 的 tab；I 的 `Meeting` | `TestK12HLSwitchTabs`、`TestDatetimePicker` |
| K12 | 一維清單：`h\|k` 往前、`j\|l` 往後；在有 tab 的 panel 裡 `h\|l` 給 tab | D「Form」；P `[3] SETTINGS`；I `[1] INPUTS` | `TestK12OneDimensional` |
| K12 | `u\|d` 半頁、`gg\|G` 頭尾；單按 `g` 不作用；其他 `g` 開頭可當熱鍵；不收 `Ctrl-U\|Ctrl-D` | P `[1] PODS`（40 列）；展示場登記一個 `[go]to` | `TestK12HalfPageAndEnds`、`TestK12SingleGNoop`、`TestK12NoCtrlUD` |
| K12 | 一步一步的移動（`j\|k`、`h\|l`、方向鍵、`Tab`）到頭尾繞回：panel 清單、一列一個值、menu、表單、finder／select／checkbox popup、panel 的 tab、picker 的欄與項目 | P `[1] PODS` 在最後一列按 `j`；tab 在最後一個按 `l`；每種清單各一次 | `TestK12StepWraps` |
| K12 | `u\|d` 到頭尾停住、不繞；照空間走的不繞：日曆的日子、月與年、左右兩側、一段文字裡的移動 | P `[1] PODS` 在倒數幾列按 `d`；I 的 `Meeting` 日曆；D「Note viewer」的 visual | `TestK12HalfPageStops`、`TestK12SpatialNoWrap` |
| K12 | 一段文字裡（viewer 的選取模式、textarea 的移動狀態）加 `w\|b\|e`、`0\|$`；viewer 平常 `h\|l` 給 tab，選取模式裡 `h\|l` 移游標、不切 tab | D「Note viewer」的 visual；D「Popup with tabs」；I 的 `Comment` 移動狀態 | `TestK12TextMotions`、`TestNoteHotkeysAndMode` |

## Rules — M Menu 與揭露

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| M1 | 非輸入態的每個畫面常駐顯示 `?` 與 `Space`（模式裡不必顯示 `Space`） | footer，每個畫面（popup 開著時照底下畫面的寫） | `TestM1FooterEntries` |
| M2 | Space menu 三區：`item operation` → `panel operation` → 一列 `Global operation`（沒熱鍵），順序固定、標題英文原字（畫成大寫，components §大寫） | 任一 panel 按 `Space` | `TestM2SectionsOrder`、`TestM2HeadingStrings` |
| M2 | global 那一列不加標題、跟上面用分隔線隔開；沒有對象就沒有那一區；item 與 panel 一律加標題（只剩一區也加） | P `[1] PODS` 的 `Empty` 之後按 `Space`（沒有 item 區） | `TestM2NoTargetNoSection`、`TestM2HeadingAlways` |
| M3 | 每個 item / panel operation 都在 Space menu；每個 global operation 都在 global operation popup；popup 的操作在 hint 與 `?` | lib：只有登記過的動作才有熱鍵 | `TestM3EveryHotkeyInMenu` |
| M3 | 只有打字列的 popup（text、password、number）：hint 列出全部操作、80 欄放得下 | I 的 `Name`、`Password`、`PIN`、`Port` | `TestM3TypingPopupHintFits80` |
| M4 | global operation popup：從 Space menu 的 global 列 `Enter`，疊在 Space menu 上；一種 menu；全部全域動作；離開在這裡 | 任一 panel：`Space` → `Global operation` | `TestM4GlobalPopup` |
| M4 | `Esc` 回到 Space menu；執行了照 T1；目前畫面的切換列停用 | 在 `[P]ANELS` 上看 `[P]anels` 那一列 | `TestM4EscBackToSpaceMenu`、`TestM4CurrentScreenDisabled` |
| M4 | global 熱鍵在 panel 上直接按（沒有 popup 時）；跟 panel 熱鍵撞鍵時 panel 的優先、那個 panel 的 key reference 不列被蓋掉的 global 鍵；數字的 global 熱鍵只在 popup 裡；模式裡不作用 | P `[1] PODS` 的 `[D]rag` 蓋掉 global 的 `[D]ialogs` | `TestM4GlobalHotkeyOnPanel`、`TestM4PanelKeyWins` |
| M4 | key reference 唯讀、可捲、沒游標、不能執行 | 任一處按 `?` | `TestKeyReferenceReadOnly` |
| M5 | 每一列：左邊動作名稱、右邊一句單行說明 | Space menu | `TestMenuRowLayout` |
| M5 | 熱鍵標記：首字母原地 `[r]ename`；中間 `UR[L]`；不在 label 裡放前面 `[n] New`；modifier `[Alt-t]erm`；多字元 `[go]to`；數字放前面 `[3] Favorites`；core key 放前面 `[Enter] Edit`；沒熱鍵不加 | 展示場的登記各用到一次（見下面「熱鍵標記」） | `TestM5HotkeyMarking` |
| M5 | 括號裡印的就是要按的鍵（大小寫算數）；label 已寫出鍵的不再括；不只靠顏色或 glyph 暗示 | 同上 | `TestM5HotkeyMarking` |
| M5 | 鍵名：鍵帽名、大駝峰（`Esc` `Tab` `Enter` `Space` `Backspace` `Delete` `Home` `End` `PgUp` `PgDn`、`↑↓←→`）；`Ctrl` 後字母大寫；`-` 連 modifier；幾個鍵做同一件事用 `\|` 並列（不用 `/`）、`–` 範圍 | lib：鍵名由一個函式產生 | `TestM5KeyNames` |
| M5 | 依位置：label 用括號；句子裡鍵加方括號；hint、footer `鍵:說明`、項目間一格；key reference 兩欄、不加括號冒號 | 空狀態、toast、hint、footer、`?` | `TestM5WritingByPosition` |
| M5 | hint、footer 的鍵一色（Blue）、冒號與說明一色（Overlay0）；並列的 `\|` 用說明的顏色（Overlay0），key reference 的鍵欄也是 | 每個 hint、footer、`?` | `TestM5HintColors`、`TestM5KeysJoinedWithBar` |
| M6 | 對象不存在：那一列（區）不出現；存在但現在不能：停用色、說明不變、cursor 跳過、熱鍵不作用 | P `[1] PODS` 的 `Disable` | `TestM6DisabledRow` |
| M6 | key reference 照同一套（停用的鍵照列、停用色）；hint 與 footer 只列現在按得了的也可以（app 決定） | `Disable` 之後按 `?` | `TestM6KeyReferenceDisabled` |
| M6 | key reference 裡講別的 surface 的段落照常顯示 | T `[1] SHELL` 的 `?` 加一段 `terminal` | `TestM6OtherSurfaceSection` |
| M7 | focus 在 panel 上時 `Space`、`?` 一定有反應；popup 上一定有反應的是 `?` | P `[2] LOG`（沒有 item）按 `Space` | `TestM7PanelEntriesRespond` |
| M8 | 其他 menu 要分組時第一組是 cursor 相關、之後每組加標題 | D「Menu: Open with」 | `TestM8GroupedMenuOrder` |
| M9 | 同鍵兩處指示同時看得到時，會觸發的亮、不會的暗 | P：`[1] PODS` 與 `[3] SETTINGS` 的下框 hint 都有 `e` | `TestM9ActiveHintBright` |

## Rules — L 版面

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| L1 | 80 × 40 能完成所有核心任務；放不下所有 panel 時照 screen 的窄寬 | 終端機開 80 × 40 走一遍；`Narrow layout` | `TestL1Core80x40`、`TestScreenNarrow` |
| L2 | 標題、statusbar、chip 等動態文字固定寬度 | statusbar 的 `Start task` 進度文字；panel 標題換 tab | `TestL2FixedWidthChrome` |
| L3 | footer、statusbar、tab 列列數鎖死；放不下截斷或從尾端捨，不折行 | 縮到 30 欄 | `TestL3ChromeRowsFixed` |
| L4 | 每一列剛好等於終端機寬度 | 每個畫面、每種 popup，多種尺寸，icon 一格與兩格 | `TestL4EveryRowFullWidth` |
| L5 | focus 不只靠顏色（粗線與細線）；切換 focus 內容不位移 | `Tab` 換 panel；Drag 模式裡照樣看得出 focus | `TestL5FocusNotColorOnly`、`TestL5NoShift` |

## Rules — F Popup

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| F1 | 七類，每類按鍵意義固定（menu、confirm、input、form、note、toast、terminal） | D 每一種各一列 | 各元件的測試 |
| F1 | note 可以有自己的熱鍵與模式；有可用 `Enter` 執行的清單就是 menu | D「Note viewer」（`/` `y` `v`） | `TestNoteHotkeysAndMode` |
| F1 | menu 的 `Enter` 可以開那一列的全文（note），仍是 menu | D「Menu: Worklist」 | `TestMenuEnterOpensFullText` |
| F1 | confirm 可以帶回答前要看的內容，長時 `j\|k` 捲 | D「Confirm: with details」「Confirm: long」 | `TestConfirmScrollsContent` |
| F1 | 依階段換類別、同一時間一類；`Tab` 切、`Esc` 關整個；預覽不拿 focus；只有有 focus 的那邊亮 | I `[/] Find` | `TestFinderPhases` |
| F1 | 清單加搜尋就是 finder（select、checkbox popup、file-picker） | I 的 `Profile`、`Namespaces`、`Identity file` | 見 finder |
| F1 | toast 不是一層（不淡化、不算層數），但 `Esc` 先收它 | D「Toast over a popup」 | `TestToastNotALayer` |
| F1 | 多步驟流程每一步是自己的 popup | D「Menu: Sort by」（欄位 → 方向） | `TestMultiStepStacks` |
| F2 | 開關都有動畫，全家族同一種：中間拉一條線、再上下展開，收起倒著播；每格 32ms、各 9 格；打開的動畫播完前不收鍵 | 每種 popup | `TestF2AnimationFrames`、`TestF2NoKeysWhileOpening` |
| F3 | `Esc` 立刻開始關閉任何 popup（含 toast）；關閉照常有動畫；在關的 popup 不再理會 `Esc` 與其他鍵 | 連按兩下 `Esc` 關兩層 | `TestF3EscImmediate`、`TestF3ClosingIgnoresKeys` |
| F3 | 例外：PTY（toast 只能等時間到）、textarea 寫入狀態、改過的表單與 textarea 先 confirm | T；I 的 `Comment`；D「Form」改過再 `Esc` | `TestF3Exceptions` |
| F4 | 從 A 開 B，A 留著；取消 B 回 A；多層一樣 | D 的疊法情境 | `TestF4KeepSource` |
| F5 | 錯誤立刻出現（toast 或 popup）、`Esc` 可關、不阻塞 app；有沒有錯誤歷史由 app 決定 | P `[1] PODS` 的 `Fail`；D「Error popup」 | `TestF5ErrorNonBlocking` |
| F6 | 要不要 confirm 由 app 決定，決定了每次都問；經 picker 明確選的可算已確認 | lib：動作登記「先問」 | `TestF6ConfirmEveryTime` |
| F6 | `Enter` 接受、`Esc` 取消；confirm 自己的熱鍵要列在 `?`；寫出接受會做的事；不能被誤觸完成 | D「Confirm: y/n」 | `TestConfirmKeys`、`TestConfirmHotkeysInHelp` |
| F7 | 寬度兩種：內容確定 = 最寬列 + 4、至少放得下標題與 hint、最多 `min(寬 − 2, 120)`；不確定 = `min(寬 − 2, 120)`；水平置中；開著不變；有整列 cursor 的，右邊的留白跟箭頭一樣寬（powerline 符號寬）；寬度至少 hint + 6（PIN 43 格） | 每種 popup；寬 200 的終端機看 120 上限 | `TestF7WidthKnown`、`TestF7WidthUnknown`、`TestF7WidthCap120` |
| F7 | 高度打開時定、最高畫面高 − 2，超過捲；打開時不知道會多高的（有預覽的 finder、textarea）一律取最高；loading 期間與使用者操作造成的改變可以變 | D「Menu: long」；I `[/] Find` 打字篩選 | `TestF7HeightFixed`、`TestF7HeightMax` |
| F7 | loading 時標題後一定有輪轉 icon；只看整個 popup 的內容 | D「Loading popup」 | `TestF7LoadingIcon`、`TestF7LoadingGrows` |
| F7 | 垂直置中；toast 的下框在畫面倒數第三列 | 每種 popup；D「Toast」 | `TestF7Position`、`TestToastPosition` |
| F7 | 值可能不合格的 popup 預留一列錯誤列；動作失敗開 error popup | I 的 `Port`；D「Form: save fails」 | `TestF7ErrorRowReserved` |
| F7 | terminal：寬 `寬 − 2`；上面留 statusbar 或 chip 列（含分隔線）、下面到最後一列蓋掉 footer；上下左右都不留白 | T `[2] SESSIONS` 的 `[Alt-t]erm` | `TestTerminalPopupSize` |
| F8 | 最上層以外全部淡化（底下 popup 與 base 畫面），一直只有最上層亮 | D 的三層疊法 | `TestF8DimAllButTop` |
| F8 | 串流內容與警示色一起淡化；淡化照 color 的算法改每一個顏色、形狀不動 | P `[2] LOG` 上開 popup | `TestF8DimStreamToo`、`TestDimEveryColor` |
| F8 | 對畫好的畫面淡化一次（失焦 panel 更暗）；toast 不觸發；底下 popup 的框是自己層色的淡化 | 同上 | `TestF8DimOnce`、`TestF8BorderKeepsLayerColor` |

## Rules — X、T、S

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| X1 | 沒有 mouse 也能完整操作 | 全部用鍵盤走過 | 整份清單的按鍵測試 |
| X2 | 有 mouse 時每個動作對應一個鍵盤動作 | lib 先不做 mouse；做的時候補 | （做 mouse 時補） |
| T1 | 預設保留 source；短 confirm、選值保留；長 session、大幅切換 context 清掉 | D 的疊法；`[2] SESSIONS` 的 `[Alt-t]erm` 從 Space menu 打開會清掉 menu | `TestT1KeepOrClear` |
| T2 | 失焦 panel 用 F8 的淡化變暗；串流內容失焦不變暗；popup 蓋上時一律淡化 | P：focus 在 `[1]`，`[2] LOG` 不暗、`[3]` 暗 | `TestT2UnfocusedDim`、`TestT2StreamNotDimmed` |
| S1 | 在 panel 上按 `V` 開 splash；某 panel 可把 `V` 給自己用，至少一個 panel 仍是 splash | P `[1] PODS` 按 `V`；I `[2] FILES` 的 `V` 給 `[V]iew` | `TestS1SplashOnV` |
| S2 | 不出現在 Space menu、global operation popup、key reference、footer、hint、README | lib：splash 沒有登記項 | `TestS2SplashNotDisclosed` |
| S3 | 任意鍵只關掉 splash（含 `q`、`Ctrl-C`、`Esc`、`Space`、`?`） | splash 上按 `Ctrl-C` | `TestS3AnyKeyClosesOnly` |
| S4 | 啟動不播；popup 開著、輸入態、PTY 裡按 `V` 不叫出 | 開著 popup、在 filter 打字、在 Shell 裡按 `V` | `TestS4SplashOnlyOnPanel` |
| S5 | 畫 icon 一格對一格、帶揭露動畫（形式 app 決定）；E8：lib 給 splash 的機制，展示場畫家族 mark | P `[1] PODS` 按 `V` | `TestS5SplashMatchesIcon` |

## Rules — E App 與環境（分工照 E8）

| 出處 | 規定 | 怎麼驗 | 測試 |
|---|---|---|---|
| E1 | 展示場：`showcase version`（`--version`、`-v`）印版號；`showcase help`（`-h`、`--help`）印命令列用法；lib 不適用 | 執行四種寫法 | `TestE1VersionHelp` |
| E2 | 變數名 `<APP>__<NAME>`；lib 讀的變數掛在 app 的名字底下；`TERMINU__ICON_WIDTH`、`TERMINU__POWERLINE_WIDTH` 給子程序 | lib 提供變數名的函式；展示場只讀 `SHOWCASE__…` | `TestE2EnvNames` |
| E3 | 設定與資料的位置：lib 不適用，展示場不存東西；家族共用的設定 key `hide_secrets_in_preview`（預設 `false`）、`input_tab_width`（預設 `4`）由 app 讀，交給 lib 的 `App.HideSecretsInPreview`、`App.InputTabWidth` | 展示場的 global `Hide secrets in preview` | `TestFilePreviewHidesSecrets`、`TestTextarea` |
| E4 | 需要 Nerd Font 與 truecolor（lib 的 README 寫明）；PUA glyph 在程式碼裡寫成 code point | 看 README；掃原始碼找不到 PUA 字元 | `TestE4NoLiteralPUA` |
| E5 | 量的是游標實際前進幾格；所有量寬度的地方走同一個函式（含疊 popup）；畫到終端機用 bubbletea v1 的 renderer（紀錄 C1） | `SHOWCASE__ICON_WIDTH=2`；`proto/c1/tmux-check.sh` 的作法 | `TestL4EveryRowFullWidth`（icon 兩格）、`TestE5OutputDropsEraseRight`（renderer 補的「清到行尾」拿掉，紀錄 C1）、`TestVTWideCharacters`（PTY 裡的字也照同一個寬度放） |
| E5 | icon 與 powerline 符號（U+E0A0–U+E0D7）各量各的；啟動時 CPR 探測（unix）；順序 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → 探測，powerline 照 `<APP>__POWERLINE_WIDTH` → `TERMINU__POWERLINE_WIDTH` → 探測；Windows 預設一格 | 在 icon 佔兩格的 tmux 裡啟動看 statusbar（寫出兩種寬度） | `TestE5WidthOrder`、`TestE5ProbeCPR` |
| E5 | 有 PTY 的 app 開子程序時設 `TERMINU__ICON_WIDTH`、`TERMINU__POWERLINE_WIDTH`（自己的格數） | T `[1] SHELL` 裡 `echo $TERMINU__ICON_WIDTH $TERMINU__POWERLINE_WIDTH` | `TestE5ChildEnv` |
| E5 | 疊 popup 不當掉：比畫面寬、高、兩邊都大都切，不 panic | 縮小終端機再開 popup | `TestE5CompositeOversized` |
| E6 | lib：git tag，沒有 binary；展示場不發布；L4 的畫面測試兩邊都要 | — | `TestL4EveryRowFullWidth` |
| E7 | lib 的 README 自己的結構（安裝 `go get`、快速開始是程式碼、寫明展示場怎麼跑；設定段寫出 E3 的共用設定 key 怎麼交給 lib）；dev-remarks 同一個骨架 | 對照這個 repo 的 README 與 `docs/dev-remarks.md` | — |
| E8 | 分工表；lib 不偏離 tdp | dev-remarks「偏離 tdp」是空的；做不到的都在紀錄裡 | — |

## color

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| color §呈現規則 1、3 | 三個錨點推導其餘；每個意義一個顏色 | lib：顏色只從語意表取 | `TestColorTokensOneMeaning` |
| color §呈現規則 2、§popup 的層色 | 第 K 層 = `lerp(Lavender, Sapphire, K/4)`，4 層以上固定；toast 一律第 1 層 | D 的四層疊法（Space menu → Global → Quit → `?`） | `TestLayerColors`（算出的值等於表 `#A4C0FA` `#94C3F5` `#84C5F0` `#74c7ec`） |
| color §呈現規則 4 | 警示色不跟層數換色，蓋在底下照 F8 淡化 | D「Confirm: with details」的 Peach 在第 3 層 | `TestWarningColorFixed` |
| color §呈現規則 6 | 粗體只用在：cursor 上的字、popup 標題、膠囊的字、模式名、標題後的 loading icon | 每種畫面 | `TestBoldOnlyWhereAllowed` |
| color §色碼 | 每個意義的色碼與用處（Base、Blue、Surface2、層色、Lavender、Green、Yellow、Peach、Red、Overlay0、Text、Subtext1、Surface1） | 每個畫面 | `TestColorTable` |
| color §色碼 | 畫面不塗底色：沒指定背景的格子留給終端機；Base 只用在色塊上的字、膠囊接縫沒填色的那一邊、淡化的目標 | 換一個不是 Base 的終端機底色看 | `TestNoBackgroundPainted` |
| color §色碼 | cursor 色塊：popup 層色底、panel Subtext1 底，字 Base 粗體；沒 focus 那塊的 cursor 再淡化一次；放在色塊上的字一律 Base | P 的 cursor；I `[/] Find` 的兩塊 | `TestCursorBlock` |
| color §淡化 | `dim(c) = c × 0.45 + base × 0.55`，前景背景都算；256 色的 16–255 照標準值；0–15 用終端機的調色盤（啟動時 OSC 4 問，問不到用 xterm 的值） | F8、T2 的畫面；在 tmux 裡改調色盤看 | `TestDimFormula`、`TestDimPaletteFromOSC4`、`TestDimPaletteFallbackXterm` |
| color §淡化 | 絕不變亮（每通道取小）；一律輸出 24-bit；沒指定前景給 `dim(Text)`；bold、reverse、游標移動、文字不動 | 同上 | `TestDimNeverBrighter`、`TestDimTruecolorOut`、`TestDimDefaultFg` |

## components §圖與間距、§大寫

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| §圖與間距 | 圖的結構是規定（框、glyph 位置、間距、對齊、標的顏色）；「縮 N 格」是離框 N 格 | 每個畫面對照規範的圖 | 各元件的版面測試 |
| §圖與間距 | 欄與欄之間兩格；glyph 與它後面的字之間一格 | 表頭、表單、menu、radio、checkbox、file-picker | `TestTwoCellColumnGap`、`TestGlyphOneCellGap` |
| §大寫 | 要大寫：panel 膠囊與 tab、畫面 chip、popup 標題、表頭、一列一個值的 label、menu 與 key reference 的區塊標題；熱鍵標記裡的字母與標題裡的名稱、路徑照原樣 | 每個畫面；`[m]ANAGE` 這種標記；`ALTERM: myhost` | `TestUppercaseWhereRequired`、`TestUppercaseKeepsHotkeyAndNames` |
| §大寫 | 不改：值、menu 的 label 與說明、hint 與 footer 的說明、按鍵名、模式名、app 的內容 | 同上 | `TestUppercaseLeavesContent` |

## layout/screen

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| screen §由上往下 | 最上面一列（多畫面：chip 列 + 全寬分隔線；單畫面：可有 statusbar）、panel 區、最下面一列 footer；每個畫面的 panel 從 `[1]` 編號 | 每個畫面；lib 的單畫面版本由測試畫 | `TestScreenRows` |
| screen §footer | 一列、M5 寫法、鍵 Blue 冒號說明 Overlay0；放不下從尾端整組捨 | 縮到 40 欄 | `TestFooterDropsWholePairs` |
| screen §footer | 一般 `Space:menu ?:help Tab\|1–N:panels q:quit`；單 panel `Space:menu ?:help q:quit`；模式 `?:help` + 模式的鍵 + `Esc:<動詞>`；filter 打字中 `Enter\|Tab:list Esc:clear`；PTY 在 panel 裡：出口鍵 + 保留鍵；popup 開著時照底下畫面的寫、跟著淡化 | P 一般與 Drag；P filter；T `[1] SHELL`；任一 popup；lib 的單 panel 版本 | `TestFooterVariants`、`TestFooterUnderPopupDimmed` |
| screen §footer | terminal popup 開著時被它蓋掉 | T 的 terminal popup | `TestTerminalPopupSize` |
| screen §畫面 chip 列 | 每個畫面一個膠囊、串的顏色 Blue（選中填 Blue 字 Base 粗體、沒選中字 Blue）；放不下先縮 `[P] [I] [D] [T]` 再收掉沒選中的 | 每個畫面；縮窄 | `TestScreenChips`、`TestScreenChipsShrink` |
| screen §畫面 chip 列 | 右邊放 statusbar；下面全寬粗線 `━` Overlay0；長工作時從左填 Green 進度條、狀態文字 Green；terminal popup 開著時 chip 列與分隔線留著 | `Start task`；開 terminal popup 時再看一次 | `TestChipRowProgress` |
| screen §statusbar | 不是 surface；label 有熱鍵就標；一般 Overlay0、你在哪 Lavender、進行中 Green、錯誤警告數 Red / Peach；寬度固定、截斷不折行 | statusbar：`icon width`、工作數、錯誤數 | `TestStatusbar` |
| screen §窄寬 | 放不下時只畫 focus 的 panel 佔滿寬、`Tab` 照換；門檻 app 定、不高於 80 | `Narrow layout`；或縮到 60 欄 | `TestScreenNarrow` |
| screen §空狀態 | 置中一句事實 + 一句點名按鍵的提示，都 Overlay0，提示的鍵加方括號；panel 的 loading 與錯誤也用這個樣子 | P `[1] PODS` 的 `Empty` | `TestEmptyState` |

## layout/panel

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| panel §框 | focus：Blue 粗線 `┏━┓`；失焦：Surface2 細直角線 `┌─┐`；同寬不位移；失焦內容淡化（串流除外） | `Tab` 換 panel | `TestPanelFrameFocus` |
| panel §框 | 上框左邊 `[N] label` 膠囊緊接框角；`N` 是跳到該 panel 的數字鍵；有 tab 時 `[N]` 自己一段、第一個 tab 的名字就是 label，在第一個 tab 時 `[N]` 與它之間一個空格（`[2] PODS`）；loading icon 放整串膠囊後、前後各空一格 | P `[1] PODS` 的 tab；`Loading` | `TestPanelTitle`、`TestPanelTabChain`、`TestPanelLoadingIconAfterChain` |
| panel §框 | 上框右邊模式名，夾在接頭之間（粗線 `┫ ┣`、細線 `┤ ├`），模式色加粗、大小寫照原樣；盡量一個詞；放不下先截標題；膠囊跟著換模式色 | P `[D]rag`；縮窄 | `TestPanelModeLabel`、`TestPanelModeLabelKeptWhenNarrow` |
| panel §框 | 下框左邊 hint（每個 panel 都有；focus 在 panel filter 打字列上、在 panel 裡的 PTY 時不寫，回來時再寫）；下框右邊捲動位置 | 每個 panel；P 按 `/` 打字；T `[1] SHELL` 按 `Enter` 進去 | `TestPanelHint`、`TestPanelNoHintWhileFiltering`、`TestK10ExitKeyShown` |
| panel §框 | 框上的字與角之間隔一段線（`┗━ Enter:logs`、`3 of 40 ━┛`、`┫Drag┣━┓`），只有 `[N] label` 膠囊緊接左上角 | 每個 panel | `TestPanelFrameTextSpacing` |
| panel §框 | hint：這個 panel 自己的鍵（`Enter` 做什麼、常用熱鍵），不寫移動鍵與 footer 已有的 core key；focus 時鍵 Blue 說明 Overlay0、失焦跟著淡化；放不下從尾端整組捨、捲動位置留著 | 每個 panel；縮窄 | `TestPanelHintContent`、`TestPanelHintDrop` |
| panel §膠囊串 | 兩端斜角 U+E0BA / U+E0B0；選中的填串色、字 Base 粗體；沒選中的不填、字用串色；斜角與接縫照 Powerline 上色（前景左段的底、背景右段的底，沒選中的底當 Base）；串色：panel 是框線色、popup 是層色、chip 是 Blue；`[N]` 膠囊與目前 tab 都算選中 | P `[1] PODS` 換 tab；D「Popup with tabs」 | `TestCapsuleChain` |
| panel §膠囊串 | 接縫：換色 U+E0B0、同色 U+E0B1；字前後不加空白、大寫；放不下先縮沒選中的成第一個字，再從尾端收成 `…`，選中的永遠留著、不撐寬框 | 縮窄 | `TestCapsuleChainShrink` |
| panel §一般清單 | cursor：Subtext1 底、Base 粗體、佔滿內寬、整列 Base 字；右邊的留白（跟箭頭一樣寬）畫 U+E0B0、Subtext1；失焦跟著淡化 | P `[1] PODS` | `TestListCursor`、`TestCursorArrow` |
| panel §一般清單 | 表頭 Blue 不加粗、大寫、固定不捲；每欄是數字或文字：數字靠右照數值排、文字靠左照文字排；欄間兩格；排序欄位加 U+F160（升）/ U+F161（降）；多欄排序在箭頭前加順位 `(2)` | P `[1] PODS` 的 `Sort by`（Name 升冪、Age 降冪、兩欄） | `TestListHeader`、`TestListSortGlyphs`、`TestListSortNumbers` |
| panel §一般清單 | 捲動位置：放不下時 `N of M`（框線色），全放得下不寫；沒有 cursor 的寫 `N-M of T` | P `[1] PODS`（40 列）、`[2] LOG` | `TestPanelScrollPosition` |
| panel §一般清單 | 標記欄：每列最前一格，標記的畫 Green U+F0132，沒標記的空白，欄寬一直保留 | P `[1] PODS` 的 `[m]ark` | `TestListMarkColumn` |
| panel §一般清單 | 截斷從尾端加 `…`；路徑、檔名可以從前面截 | P `[1] PODS` 的長名字；I `[2] FILES` 的路徑 | `TestListTruncate` |
| panel §一般清單 | 空的照空狀態 | `Empty` | `TestEmptyState` |
| panel §loading 與錯誤 | loading：icon 在整串膠囊後；沒內容時中間寫一句（`󰪞 loading pods…`）；重新載入舊內容留著 | `Loading`（空的時候與有內容時各一次） | `TestPanelLoading` |
| panel §loading 與錯誤 | 錯誤：置中兩句，事實 Red、提示 Overlay0；舊內容在就留著、錯誤用 toast 報 | `Fail`（空的時候與有內容時各一次） | `TestPanelError` |
| panel §一列一個值 | 照表單：項目逐一移動（有 tab 時 `h\|l` 給 tab）；`Enter` 開 input popup，radio、checkbox 原地選；確定後 focus 留在同一列 | P `[3] SETTINGS`；I `[1] INPUTS` | `TestValuePanelMoves`、`TestValuePanelEnter` |
| panel §一列一個值 | 沒有送出：確定的那一刻就生效；沒有動作列、沒有「改過了要先問」 | 同上 | `TestValuePanelAppliesNow` |
| panel §一列一個值 | focus 色塊 Subtext1 底 Base 粗體、最右邊一格是箭頭，從項目開頭到 panel 內右緣；label 大寫，一般 Blue、停用 Surface2、出錯 Red；label 欄寬取最長的 label、與值之間兩格 | `[3] SETTINGS` 的 `Disable`、`Fail` | `TestValuePanelFocusBlock`、`TestValuePanelLabels` |
| panel §panel filter | `/` 在最上面出現打字列、即時篩選、不開 popup；最前面 U+F105 跟清單的字對齊（離框一格）、空一格才接打的字，打字時框線色（Blue）、在清單時 Overlay0；右邊 `符合/總數`，Overlay0；打字列按鍵與比對同 finder；篩選後保持 panel 的排序 | P `[1] PODS` 按 `/` | `TestPanelFilter` |
| panel §panel filter | 清單上 `Tab` 下一個 panel；清單上 `Esc` 清掉篩選；選了之後清單留著篩選、打字列變灰；清單上 `/` 回打字列接著打 | 同上 | `TestPanelFilterListKeys` |
| panel §panel filter | focus 離開篩選留著；回來時直接在打字列上（`Tab` 經過停兩下） | 同上 + `Tab` 繞一圈 | `TestPanelFilterFocusReturn` |

## layout/popup

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| popup §外框 | 粗線 `┏━┓`、層色；被蓋在底下照樣是粗線 | 每種 popup | `TestPopupFrame` |
| popup §外框 | 標題：glyph + 文字做成膠囊，放在 `┏━` 後；兩端圓角 U+E0B6 / U+E0B4；填層色，字 Base 粗體、大寫（名稱與路徑照原樣）；太長從尾端截加 `…` | 每種 popup；D「Popup: long title」 | `TestPopupTitleCapsule`、`TestPopupTitleTruncate` |
| popup §外框 | 標題後接 tab 照膠囊串畫（跟圓角的標題之間隔一段線） | D「Popup with tabs」 | `TestPopupTitleTabs` |
| popup §外框 | 標題 glyph：menu U+F0C9、key reference U+F059、confirm U+F0ECC、toast 依種類、terminal U+F018D、input／表單／note 由 app 選 | 每種 popup | `TestPopupTitleGlyph` |
| popup §外框 | hint 放 `┗━ ` 後；順序 `Enter` → 自己的鍵 → `Tab:…` → `Esc`（terminal 出口鍵第一）；不寫移動鍵（換塊的照寫）；放不下從尾端整組捨；只有打字列的列全部、80 欄放得下 | 每種 popup | `TestPopupHintOrder`、`TestPopupHintDrop` |
| popup §外框 | 框上的字與角之間隔一段線：`┏━TITLE`、`┗━ Enter:run`、`4 of 12 ━┛` | 每種 popup | `TestPopupFrameTextSpacing` |
| popup §外框 | 內容上下各留一列、左右離框一格（terminal 上下左右都不留） | D「Menu: long」 | `TestPopupMargins` |
| popup §外框 | 清單的 cursor 從框貼到框、字照樣離框一格；表單的 focus 色塊從值到框；右邊的留白是 U+E0B0、色塊的顏色，跟箭頭一樣寬；只有整列的 cursor 有箭頭 | Space menu；D「Form」 | `TestPopupCursorCoversPadding`、`TestCursorArrow` |
| popup §外框 | 分隔線不接框：橫的左右各留一格、直的不畫進上下留白；Overlay0 | D「Menu: long」；I 的 `Meeting` | `TestPopupSeparator` |
| popup §寬度 | 兩種寬度（見 F7）；「確定」= 打開那一刻就知道最寬（PIN 一開就 27 格） | I 的 `PIN` | `TestF7WidthKnown` |
| popup §寬度 | 窄的內容靠左離框一格；toast 文字與 password 解鎖遮罩置中 | D「Confirm」、「Toast」；I 的 `PIN` | `TestPopupNarrowContentAlign` |
| popup §開關動畫 | 拉線 4 格（正中間那一列的層色 `━` 從中間往左右拉到 popup 寬）→ 停 1 格 → 展開 4 格（從中間往上下露出）；收起倒著播；每格 32ms | 每種 popup，含 toast 與 terminal | `TestF2AnimationFrames` |
| popup §開關動畫 | 打開的第一格起底下就淡化；開始收起時底下立刻亮回來；打開的動畫播完前與收起時不收鍵 | 同上 | `TestF2DimFromFirstFrame`、`TestF2NoKeysWhileOpening` |
| popup §loading | icon U+F0A9E–U+F0AA5 八格；一格 90ms；格數由時鐘決定 `frames[(now / 90ms) % 8]`；只在有東西 loading 時續排 tick；一格寬、換上換下不位移；顏色跟旁邊字、popup 標題用層色粗體；放在整串 tab 後、前後各空一格 | D「Loading popup」；P `Loading` | `TestLoadingIconFrames`、`TestLoadingTickOnlyWhenLoading` |
| popup §取消與完成 | 取消回 source；完成照 T1 | D 的疊法 | `TestF4KeepSource`、`TestT1KeepOrClear` |
| popup §內容放不下時 | 只有內容區捲，留白、錯誤列、上下框固定；跟著 focus 捲最少的量；`N of M`（有 cursor）／`N-M of T`（note），層色；有游標的（textarea、選取模式裡的 viewer）也寫 `N of M`，以行為單位；表單以欄位為單位、只算欄位，focus 在錯誤列或按鈕上時不寫 | D「Menu: long」、「Note viewer」、「Form: long」 | `TestPopupScroll`、`TestPopupScrollPosition`、`TestFormScrollPosition` |
| popup §錯誤列 | 內容區下面、空一列；只有值可能不合格的預留；只放值的錯；Red、左緣對齊（內容置中時也置中）；框與標題維持層色 | I 的 `Port` | `TestErrorRow` |
| popup §錯誤列 | 只佔一列、截斷加 `…`；表單上可 focus、`Enter` 開完整訊息；input popup 裡不拿 focus；只在檢查時改 | D「Form」送出失敗；I 的 `Port` | `TestErrorRowTruncate`、`TestErrorRowChangesOnCheckOnly` |
| popup §錯誤列 | 動作失敗開 error popup 蓋在上面，關掉回原本的 popup、值都在 | D「Form: save fails」 | `TestActionFailureErrorPopup` |

## dialog/form

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| form §用途 | 表單上不打字：要打字或選項放不下的欄位 `Enter` 開 input popup；radio、checkbox 畫在表單上原地選 | D「Form: New host」 | `TestFormNoTyping` |
| form §標題 | 寫表單要做什麼（標題膠囊），不寫型別 | 同上 | — |
| form §欄位的排法 | 上下式、左右式，可混用；欄位之間、label 與值之間不空行；label、值、錯誤左緣對齊；左右式 label 欄寬取左右式最長的、與值之間兩格 | D「Form: New profile」（混用） | `TestFormLayouts` |
| form §選項畫在表單上 | 五個以內畫在表單、超過用 select／checkbox popup；radio 與 checkbox 組一律直排、一個選項一列 | D「Form」的 `Auth`、`Order` | `TestFormOptionsDrawn` |
| form §focus 與 label | focus 照 menu 的 cursor（層色底 Base 粗體），從項目開頭一直到框（蓋住右邊留白）、最右邊一格是箭頭；按鈕的色塊就是按鈕、沒有箭頭；值空著也一整塊 | 同上 | `TestFormFocusBlock` |
| form §focus 與 label | label 不因 focus 變色：一般層色、停用 Surface2、值出錯 Red（優先）；停用的留原位；值不跟 label 同色 | D「Form」的 `Credential`（停用）；送出失敗 | `TestFormLabelColors` |
| form §動作列 | 按鈕：文字左右各一格、一塊底色；沒 focus Surface1 底 Text 字、focus 層色底 Base 粗體；寫要做的事；不用 `[ Save ]`、不用膠囊 | D「Form」 | `TestFormButton` |
| form §動作列 | 位置：欄位區 → 空一列 → 錯誤列 → 分隔線 → 動作列；按鈕靠右離框一格；錯誤列、分隔線、動作列固定不捲；只有一個按鈕；沒有取消按鈕 | D「Form: long」（欄位要捲） | `TestFormActionRowFixed` |
| form §移動 | 項目：欄位、每個選項、按鈕、有錯誤的錯誤列（在按鈕前） | D「Form: New host」 | `TestFormItems` |
| form §移動 | `j\|l\|↓\|→` 下一個、`h\|k\|↑\|←` 上一個；跨欄；`gg` 第一個、`G` 按鈕；`u\|d` 以項目為單位半頁、到頭尾停住；`Tab\|Shift-Tab` 一次一欄；頭尾繞回 | 同上 | `TestFormMoveItems`、`TestFormTabStops`、`TestFormWrap`、`TestFormHalfPage` |
| form §移動 | 跳進 radio 停在選中的（都沒選停第一個）、checkbox 組停第一個，落點跟方向無關 | 同上的 `Auth`、`Agent` | `TestFormTabStops` |
| form §移動 | 停用的跳過；打開時 focus 在第一個能停的（從某一欄打開就停那欄）；不自動打開第一欄 | D「Form: open at Port」 | `TestFormSkipDisabled`、`TestFormOpenFocus` |
| form §`Enter` | 欄位：開 input popup 疊在表單上，確定寫回 focus 留原欄，`Esc` 值不變；checkbox popup：`Space` 勾、`Enter` 寫回、`Esc` 取消 | 同上的 `Name`、`Namespaces` | `TestFormEnterField` |
| form §`Enter` | radio 選項：選它；checkbox 選項：原地翻；都不開 popup、focus 不動、馬上生效；按鈕：送出 | 同上 | `TestFormEnterOption`、`TestFormEnterButton` |
| form §送出 | 只有按鈕的 `Enter`；不用字母熱鍵；`Esc` 不把 focus 移到動作列 | 同上 | `TestFormSubmitOnlyButton` |
| form §在欄位上打字 | 要打字的欄位上按表單操作鍵以外的可列印字元（含數字、`q`）跳說明 note：標題是欄位名、`Enter`／`Esc` 關、focus 不動；其他項目上不跳 | D「Form」的 `Name` 上打 `s` | `TestFormTypingNote` |
| form §在欄位上打字 | 表單操作鍵不跳 note：移動鍵、`u\|d`、`g`（單按不作用）、`G`、`e`（沒錯時不作用）、`Enter`、`Esc`、`Space`（不作用）、`?`（key reference） | 同上，按 `e` `u` `g` `?` | `TestFormOperationKeys` |
| form §hint | `Enter:` 後的字跟著 focus 換（`edit`／`choose`／`toggle`／按鈕小寫）；有問題的欄位加 `e:error`；錯誤列 `Enter:show e:edit Esc:cancel`；不寫移動鍵；`Esc:cancel` | 同上 | `TestFormHint` |
| form §取消 | 沒改過 `Esc` 直接關；改過先 confirm（`Enter` 放棄、`Esc` 回表單值都在）；「改過了」只要寫回過就設、不清掉；只有表單與 textarea 問 | D「Form」改一欄再 `Esc` | `TestFormDiscardConfirm`、`TestFormDirtyFlag` |
| form §錯誤 | 一欄自己的規則在 input popup 擋；整張表單的規則送出時檢查；沒過：不存、每個有問題的 label Red、錯誤列寫第一個、focus 跳過去 | D「Form」空著送出、Port 打 `0` | `TestFormValidation` |
| form §錯誤 | `e`：有問題的欄位 → 錯誤列換成這一欄的訊息；錯誤列 `Enter` 開完整訊息的 error popup（標題欄位名、`Enter`／`Esc` 關）；錯誤列 `e` 回那一欄 | 同上 | `TestFormSubmitChecks` |
| form §錯誤 | 送出動作失敗：error popup，關掉回表單、focus 在按鈕；錯誤跟著檢查走、第一次送出前不檢查 | D「Form: save fails」 | `TestFormSaveFails`、`TestFormNoCheckBeforeSubmit` |

## dialog/menu

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| menu §長相 | label Text、`[x]` 同色；說明 Overlay0 靠右離右框一格、太長截加 `…`；label 與說明之間最少兩格 | Space menu；D「Menu: long descriptions」 | `TestMenuRowLayout` |
| menu §長相 | 區塊標題：U+F02D8（六角形）+ 一格 + 大寫標題，Overlay0、離框一格、不加粗、cursor 跳過；區塊之間的分隔線 `╌`（U+254C）、不接框、Overlay0、cursor 跳過 | Space menu | `TestMenuHeadings`、`TestMenuSeparator` |
| menu §長相 | cursor 層色底、Base 粗體，從框貼到框、字離框一格、最右邊一格是箭頭；停用的列整列 Surface2、cursor 跳過、熱鍵不作用 | `Disable` 之後的 Space menu | `TestMenuCursor`、`TestM6DisabledRow` |
| menu §長相 | 標題 U+F0C9 + 文字（標題膠囊）：Space menu 寫 focus panel 的 `[N] label`、其他寫它做什麼（`GLOBAL OPERATION`、`SORT BY`）；`Global operation` 列說明一律 `actions for the whole app`；寬度照「內容確定」 | Space menu、Global operation、Sort by | `TestMenuTitle`、`TestMenuGlobalRow` |
| menu §按鍵 | `j\|k` 到頭尾繞回；`u\|d`（到頭尾停住）、`gg\|G`；`Enter` 執行、熱鍵直接執行；`Esc` 關（global 回 Space menu）；Space menu 再按 `Space` 關；hint `Enter:run Esc:close`；打開那一列全文的 menu（工作清單）寫 `Enter:detail Esc:close` | 同上 | `TestMenuKeys`、`TestMenuWraps`、`TestMenuEnterOpensFullText` |

## dialog/confirm

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| confirm §長相 | 標題 U+F0ECC + 動作名（大寫）；問句放最後、下面接 hint；內容放問句上面、空一列 | D「Confirm」「Confirm: with details」 | `TestConfirmLayout` |
| confirm §長相 | 問句 Text 不加粗、內容 Overlay0、警告那一句 Peach（哪句由 app 定） | D「Confirm: with details」 | `TestConfirmColors` |
| confirm §長相 | 太長折行不截；內容長時只有內容 `j\|k` 捲、問句固定；寬度照「內容確定」 | D「Confirm: long」 | `TestConfirmWrapAndScroll` |
| confirm §按鍵 | `Enter` 接受、`Esc` 取消、自己的熱鍵列在 `?`、`Space` 不作用；hint `Enter:<動詞> Esc:cancel` | D「Confirm: y/n」 | `TestConfirmKeys`、`TestConfirmHint` |

## dialog/note

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| note §長相 | 留白一律有；標題 glyph（app 選）+ 顯示的東西（標題膠囊，`YAML — pod/nginx`）；捲動位置 `N-M of T`（viewer 的選取模式裡 `N of M`）；寬度：確定的照內容、viewer 用上限 | D「Note viewer」 | `TestNoteLayout` |
| note §按鍵 | 捲動 `j\|k` `u\|d` `gg\|G`；只用 `Esc` 關（`Enter`、`Space` 不關）；error popup（不管怎麼打開的）與表單的打字說明 `Enter` 也能關；hint 熱鍵在前、`Esc:close` 最後 | 同上；D「Error popup」 | `TestNoteKeys`、`TestNoteHint` |
| note §選取模式 | 只有 viewer 有；平常沒有游標（`j\|k` 捲、有 tab 時 `h\|l` 切 tab）；第一次 `v` 進模式（框與模式名 Yellow），游標在畫面第一行行首、還沒選，hint `v:select Esc:cancel`；`h\|j\|k\|l` `w\|b\|e` `0\|$` `u\|d` `gg\|G` 移游標、畫面跟著捲；第二次 `v` 從游標開始選（以字為單位、含游標那個字），hint `y:copy Esc:cancel`，再按 `v` 不作用；`y` 複製、整個離開；`Esc` 在哪一步都整個離開、不複製；游標層色底 Base 字，範圍 Yellow 底 Base 字、空行畫一格；捲動位置 `N of M`（游標的行） | D「Note viewer」按 `v` | `TestNoteHotkeysAndMode`、`TestViewerSelectScrollPosition` |
| note §key reference | 標題 U+F059 + `<surface> keys`（`[2] HOSTS KEYS`、`SPACE MENU KEYS`）；每列一個鍵：縮兩格、鍵補到最寬、空兩格、說明（截斷不折行）；從 menu 來的列 `label — 說明`、label 去掉標記、大小寫照原樣 | 每個 surface 按 `?` | `TestKeyReferenceLayout` |
| note §key reference | panel 上的區塊依序：`item operation`、`panel operation`、`move`（真的能用的移動鍵）、`core keys`（`Space`、`?`、`Tab`、`1–N`、`q\|Ctrl-C`；`Esc` 只在有作用時）、`app-wide`；區塊標題同 menu | P `[1] PODS`（有 tab：列 `h\|l`）、`[2] LOG`、有篩選時（列 `Esc`） | `TestKeyReferenceSections`、`TestKeyReferenceMoveSection`、`TestKeyReferenceCoreKeys` |
| note §key reference | popup 上只列它的熱鍵、`Enter`、`Esc`、移動鍵；鍵 Blue 不加粗、說明 Text、按不了的 Surface2；`?`／`Esc` 關、`Enter` 不作用；hint `?\|Esc:close` | Space menu 上按 `?` | `TestKeyReferenceKeys` |
| note §error popup | 框、文字 Red；標題膠囊填 Red、字 Base 粗體；標題 U+F0238 + 哪件事失敗（表單錯誤列打開的寫欄位名）；訊息折行、太長 `j\|k` 捲 | D「Error popup」「Form: save fails」 | `TestErrorPopupLayout` |
| note §error popup | `Enter`、`Esc` 都能關，hint `Enter\|Esc:close`；關掉回底下的 popup、focus 原位、值都在 | 同上 | `TestErrorPopupClose` |

## dialog/terminal

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| terminal §用途 | PTY 在 panel 裡時不是 popup：panel 的框、鍵寫在 footer；focus 在 PTY 裡時 panel 下框不寫 hint | T `[1] SHELL` | `TestK10KeysToChild`、`TestK10ExitKeyShown` |
| terminal §長相 | 大小照 F7 的例外（chip 列與分隔線留著）；沒有 statusbar 的 app 從第一列開始；上下左右都不留白；標題 U+F018D + 跑的是什麼；子程序的游標畫成一格 Text 底、Base 字的方塊，DECSCUSR 要的形狀不理（panel 裡也一樣） | T `[2] SESSIONS` 的 `[Alt-t]erm` | `TestTerminalPopupSize`、`TestTerminalCursorBlock` |
| terminal §按鍵 | 按鍵都給子程序；hint 出口鍵第一、最後才捨；`Alt-Esc:exit`（結束）／`Alt-Esc:leave`（留著），不寫 `close`；後面跟保留鍵；`Alt-Esc` 先 confirm | 同上 | `TestK10ExitKeyShown`、`TestK10AltEscConfirms`、`TestEncodeKey` |

## dialog/toast

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| toast §長相 | popup 的框、一律第 1 層層色；不觸發淡化、也不被淡化 | D「Toast over a popup」 | `TestToastNotALayer` |
| toast §長相 | 畫面下方水平置中；下框一律在畫面倒數第三列，最下面兩列一直看得到（平常是 panel 下框與 footer；terminal 開著時是 PTY 最後一列與 terminal 下框）；寬度照「內容確定」 | `Toast Info`；T 的 terminal popup 裡跳 toast | `TestToastPosition`、`TestToastPositionOverTerminal` |
| toast §長相 | 文字置中、太長折行每列置中、最多三列、超過第三列結尾 `…` | `Toast long` | `TestToastWrapThreeLines` |
| toast §長相 | Info U+F024 `INFO` Text 字；Warning U+F078F `WARNING` Peach；Error U+F0238 `ERROR` Red | `Toast Info` / `Warning` / `Error` | `TestToastKinds` |
| toast §長相 | Info 2200ms，Warning、Error 4400ms；第二個來直接取代、重新計時 | 連按兩次 `Toast …` | `TestToastDuration`、`TestToastReplace` |
| toast §按鍵 | `Esc` 立刻收、比其他 popup 先收；其他鍵穿過；PTY 裡只能等時間到；hint `Esc:close` | toast 開著時按 `j`；T 裡跳 toast | `TestToastEscFirst`、`TestToastKeysPassThrough` |

## input/README（所有 input 共用）

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| input §什麼是 input | 打字列上是輸入態；不打字的部分 core key 照 K1（`?` key reference、`q`／`Ctrl-C` 離開、`Space` 不作用、`hjkl` 移動）；`Enter` 照 K3；`Esc` 關值不變；從表單開的確定後 focus 留原欄 | I `[1] INPUTS` 每一列 | `TestInputCommon` |
| input §單行骨架 | 說明列一句話 Overlay0、空一列；值 Lavender、游標 Lavender 底 Base 字；錯誤列；hint `Enter:<動詞> Esc:cancel`（列出全部操作）；寬度：自由打字用上限、有長度上限照內容 | I 的 `Name` | `TestSingleLineSkeleton` |
| input §值比框長時 | 顯示游標附近那段、截掉的那邊補 `…`；游標一定看得到、只捲剛好的量；照顯示寬度、一個字不切半（紅色 `\n` 也是） | I 的 `Path`（長路徑），`Home`／`End` | `TestInputLongValue` |
| input §編輯鍵 | 字元、`Space` 插入；`←\|→`；`Home\|End`；`Backspace`、`Delete`；一個字是看起來的一個字；不收 `Ctrl-U`、`Ctrl-W`；其他 `Ctrl-`、`Alt-` 不作用不打字 | I 的 `Name`（打中文、emoji、組合字元） | `TestInputEditKeys`、`TestInputGraphemes`、`TestInputCtrlAltIgnored` |
| input §灰字提議 | 只用在單行（text、number）；打開是空的、舊值灰字在游標後；`Tab` 收進來；直接打字換新值（刪光再出現）；空的時候 `Backspace` 丟掉舊值、`Enter` 存空；什麼都沒碰 `Enter` 不改；hint 加 `Tab:edit Backspace:clear`；只有舊值這一種來源 | I 的 `Name` | `TestInputSuggestion` |
| input §不放 placeholder | input popup 裡不放 placeholder | I 每一列 | `TestInputNoPlaceholder` |
| input §貼上的換行與 Tab | 一行的值：貼上的換行與 Tab 留著、畫紅色 `\n` `\t`（2 格不切）、灰列裡跟著灰；`\r\n` 算一個；C0、DEL、C1 丟掉；`Backspace` 整個刪；遮罩照遮 | I 的 `Name` 貼上多行 | `TestPasteMarkers`、`TestPasteControlDropped` |
| input §貼上的換行與 Tab | 不是打字來的值走同一個過濾；會被拿去用的在 `Enter` 時擋、錯誤列點名欄位；在去空白字元之前檢查；搜尋與篩選的打字列只畫不擋 | I 的 `Name`；I `[/] Find` 貼上 | `TestPasteBlockedOnEnter`、`TestPasteSearchNotBlocked` |
| input §值 | 一行的值確定時去掉前後的空白字元（所有 Unicode 空白：一般空格、U+3000、NBSP…）；password 不去 | I 的 `Name` 打 `　x ` | `TestInputTrimWhitespace` |
| input §在表單與 panel 上怎麼顯示 | Text；空值灰字寫「空著代表什麼」（app 寫、不教按鍵）；太長尾端截、路徑前面截；各 input 的顯示（password 8 個 `•`、textarea 第一行 + `…`、checkbox popup 用 `, ` 接、color 色塊 + hex、file `~`…） | I `[1] INPUTS` 每一列（含空的） | `TestValueDisplay` |

## input — 各種

| 出處 | 規定 | 展示場 | 測試 |
|---|---|---|---|
| text | 單行骨架、編輯鍵、提議、貼上照 README；確定去空白字元 | I 的 `Name`、`Path` | `TestTextInput` |
| password §長相 | 兩個方案：一般（每字一個 `•` 靠左）、解鎖（每字一顆 `●` 間隔一格、置中往兩側長）；遮罩 Lavender；舊密碼 8 個灰點用同方案的字；放不下照「值比框長時」；有長度上限照內容寬 | I 的 `Password`（一般）、`PIN`（解鎖，12 碼 27 格） | `TestPasswordMasks`、`TestPasswordUnlockWidth` |
| password §狀態 | 打開是空的、有舊密碼畫 8 個灰點；什麼都沒打 `Enter`／`Esc` 保留；打字換新的；沒有 `Tab` | I 的 `Password`（已設） | `TestPasswordExisting` |
| password §按鍵 | 貼上照 README（照遮）；`Backspace` 一次清全部（打的 → 回到灰點；空的 → 清掉舊密碼、`Enter` 存成沒有）；hint 加 `Backspace:clear`；只接在尾端，`←→` `Home` `End` `Delete` 沒作用 | 同上 | `TestPasswordBackspaceClears`、`TestPasswordAppendOnly` |
| password §值 | 不去空白字元；表單與 panel 上設了是 8 個 `•`（Text）、沒設照空值 | I `[1] INPUTS` 的兩列 | `TestPasswordDisplay` |
| number | 說明列寫範圍；範圍已知寬度確定；打字只收可能的字（負數 `-`、小數 `.`）、貼上只留會收的；不做 `↑↓` 加減；範圍在 `Enter` 檢查 | I 的 `Port`（1-65535）、`Ratio`（小數、可負） | `TestNumberFilter`、`TestNumberRange` |
| textarea §長相 | 太長自動折行、不左右捲；行號每行第一列、靠右、寬度取最大行號位數、後空一格、折下來的空白；行號 Overlay0、游標那行 Text；寬度與高度都用上限、內容在框裡捲，放不下時捲動位置寫游標的行 `N of M`；值裡的 `\t` 畫成 `input_tab_width` 格 | I 的 `Comment` | `TestTextareaWrapAndNumbers` |
| textarea §按鍵 | 寫（輸入態、`Enter` 換行、`Tab` 插 `input_tab_width` 個空白（設定、預設 4）、hint `Esc:move`、游標 Lavender 一格）／移（游標層色一格；`Esc` 進來、`i` `a` `o` 回寫、`Enter` 確定、hint `Enter:save i:write Esc:cancel`）；移裡 `Esc` 關（改過先 confirm）；打開在「寫」、舊值帶入、游標在最後、沒有灰字提議 | 同上（有舊值） | `TestTextarea` |
| textarea §按鍵 | 寫的編輯鍵加 `↑↓`（照畫出的列）、`←→` 跨行、行首 `Backspace` 合併；移的鍵 `hjkl` `w\|b\|e` `0\|$` `u\|d` `gg\|G`；移不是模式也不是輸入態（`?` 開 key reference） | 同上 | `TestTextareaWriteKeys`、`TestTextareaMoveKeys` |
| finder §長相 | 兩塊：打字列（最前面 U+F105、離框一格、空一格接打的字；focus 在打字列時層色、在清單時 Overlay0；右邊 `符合/總數`，Overlay0）、分隔線、清單；打字列的字跟清單的字對齊；清單的 cursor 從框貼到框 | I `[/] Find` | `TestFinderLayout` |
| finder §長相 | 一般 finder 一定有預覽（select、checkbox popup 沒有）；兩框合起來一個 popup 寬；高度一律取最高；寬 ≥ 96 左右並排各一半（有 metadata 時右框是預覽／分隔線／四列 metadata）；不到 96、有 metadata：上面是清單框與 `INFO` 框各一半、下面是整條寬的預覽框，上下各一半；不到 96、沒有 metadata：清單在上、預覽在下，各一半；預覽標題是那一筆的名字、不拿 focus；預覽檔案或目錄的 finder 照 file-picker 的預覽，其他的內容 app 給、沒有 metadata；預覽框用層色、內容不淡化、下框不寫字 | I `[/] Find`，寬 120 與 80 各看一次 | `TestFinderPreview`、`TestFinderFileLayout` |
| finder §長相 | 邊找邊列：標題後 loading icon、一筆都沒到寫 `(indexing…)`；高度打開時定、寬度用上限 | I `[/] Find` 的 `Slow index` | `TestFinderStreaming` |
| finder §清單的標記 | select 前面 radio glyph、checkbox popup 前面 checkbox glyph；目前的值字 Green；cursor 在目前的值上照 cursor | I 的 `Profile`、`Namespaces` | `TestFinderMarkers` |
| finder §狀態 | 只有 focus 那塊亮：focus 在打字列時清單 cursor 是層色 cursor 再淡化；focus 在清單時打字列整列灰、沒游標 | I `[/] Find` 按 `Tab` 來回 | `TestFinderFocusBrightness` |
| finder §按鍵 | 打字列：字元篩選、`↑↓` 移 cursor（繞回）、`Enter` 到 cursor 那項（沒結果不作用）、`Tab` 到清單、`Esc` 關整個（filter 是清掉）；清單：移動（繞回）、`Enter` 去那裡、`Tab\|/` 回打字列接著舊的字打、`Esc` 關整個 | 同上 | `TestFinderTypingKeys`、`TestFinderListKeys` |
| finder §比對 | 包含、不分大小寫、字要連續出現；空白分開的每個詞都要出現、順序不拘；結果保持清單本來的順序（數字照數值、文字照文字）；用在 finder、select、checkbox popup、file-picker 的一般字、panel filter | I `[/] Find` 打 `read`、`READ me`；P filter 打 `run 3d` | `TestMatchContains`、`TestMatchWords`、`TestMatchKeepsOrder` |
| finder §hint | 打字列 `Enter\|Tab:list Esc:cancel`；清單 `Enter:<動詞> Tab\|/:filter Esc:cancel` | 同上 | `TestFinderHint` |
| select | finder 骨架沒預覽、打字列一直顯示；radio glyph、目前的值 Green；選項可有熱鍵（app）；寬度照選項；打開 focus 在清單、cursor 在目前的值；`Enter` 選定寫回關掉（從表單開的 focus 留原欄）；hint 清單 `Enter:choose Tab\|/:filter Esc:cancel`、打字列 `Enter\|Tab:list Esc:cancel` | I 的 `Profile`（帶數字熱鍵） | `TestSelect` |
| radio | 五個以內畫在表單或 panel；U+F043E／U+F043D；選中 Green 其他 Text；一律直排、一個選項一列；每個選項一個項目；`Enter` 選、focus 不動、馬上生效 | I 的 `Layout`、`Order` | `TestRadio` |
| checkbox §長相 | U+F0132（勾）／U+F0131；勾的字 Green；popup 是 finder 骨架沒預覽、checkbox glyph、寬度照選項 | I 的 `Notify`（開關）、`Agent`（一組）、`Namespaces`（popup） | `TestCheckbox` |
| checkbox §按鍵 | 表單或 panel 上 `Enter` 原地翻；popup：打開在清單、`Space` 只改畫面、`Enter` 整組寫回才生效（從表單：寫回並標改過；單獨：這時才套用；沒改就關）、`Esc` 取消不問；勾的字跟著畫面上的勾變 Green；hint `Enter:apply Space:toggle Tab\|/:filter Esc:cancel` | I 的 `Namespaces`；D「Form」的 `Namespaces` | `TestCheckboxPopup` |
| slider | 表單與 panel 上軌道 `━` + `●` + 數字（寬度與顏色 app 定）；`Enter` 開 select（bottom 到 top、每 step 一個，cursor 在目前值，打數字篩選）；沒有自己的鍵；bottom、top、step 必有，step 預設 `(top − bottom) / 10`、可以是小數 | I 的 `Brightness`（0–100）、`Opacity`（0–1，step 0.05） | `TestSlider` |
| datetime §日期 | 月、年各是可 focus 的項目（focus 時照 cursor）、之間兩格；日曆靠左離框一格、不另外縮；每天 4 格（日期 2 格靠右 + 間距 2）；固定 6 列；cursor 那天就是要寫回的那天；原本的值 Green、打開 cursor 在它（沒有就今天）；今天底線；週末不上色；一週從週一或週日（app）；範圍外 Surface2、cursor 不停 | I 的 `Due date`（有範圍、週一開始） | `TestDatePickerLayout` |
| datetime §日期加時間 | 右邊直分隔線（不接框、左右各空兩格）；時 00–23、分每 5 分、兩欄之間兩格；目前的值 Green；跟日曆一樣高、超出捲；值不在格上停在之前最近的；沒 focus 那塊與時間塊裡沒 focus 那一欄的 cursor 再淡化（整個 picker 只有一個亮的 cursor）；版面固定、寬度確定 | I 的 `Meeting`（值 10:07） | `TestDatetimePickerLayout`、`TestDatetimeOneBrightCursor` |
| datetime §按鍵 | `Tab`：月 → 年 → 日曆 → 時間 → 月，打開在日曆；月年 `h\|l` 換、日曆跟翻（沒那天停最後一天）；日曆 `h\|l` 一天 `j\|k` 一週、跨月、跳過範圍外、不繞；時間 `j\|k` 欄內（繞回，各欄自己繞）、`h\|l` 換欄；`Enter` 各塊；`t` 今天（時間到現在、往前取 5 分）；`Esc` 不改；hint 五種 | 同上 | `TestDatetimePicker`、`TestDatetimeTimeWraps`、`TestDatetimeEnter`、`TestDatetimeHint` |
| datetime §值 | 可設最早最晚；表單與 panel 上格式 app 定，空照空值 | I `[1] INPUTS` | `TestDatePickerRange` |
| color-picker §長相 | 一列：old、new、R、G、B，五個項目之間兩格；old／new 色塊 8×2 + 名字 + hex；R、G、B 色塊 3×1（畫該色版自己的顏色）+ hex；下面分隔線與 `Choose` 按鈕（照表單的按鈕）、沒有錯誤列；寬度確定 | I 的 `Background` | `TestColorPickerLayout` |
| color-picker §按鍵 | 可 focus：R、G、B、`Choose`（`h\|l`、`j\|k` 前後、到頭尾繞回；old、new 不拿 focus）；打開在 R；focus 的色版下面兩列（名字與 hex）照 cursor、跟色塊一樣寬；R/G/B `Enter` 開 select 00–FF（`2A (42)`、hex 十進位都能篩），回來 new 跟著變、focus 留著；`Choose` 上 `Enter` 確定；`Esc` 不改；hint 兩種 | 同上 | `TestColorPickerKeys`、`TestColorPickerChannelSelect` |
| color-picker §值 | 表單與 panel 上一小格色塊 + hex | I `[1] INPUTS` | `TestValueDisplay` |
| file-picker §長相 | 根目錄 app 給、只是起點，最上面 `..`（`/` 沒有）；目前的目錄寫在標題（欄位名後，太長從前截）；打字列打開是空的；每列前目錄／檔案 icon（顏色 app 定）；原本的值 cursor 停在它、字 Green；`..` → 目錄 → 檔案、照名字不分大小寫，隱藏檔只在篩選字 `.` 開頭時列（`*` 不比對隱藏檔、`**` 不走進隱藏目錄）；打字列前面 U+F105；四塊：打字列、清單、預覽、metadata；預覽最多讀 64 KB：目錄與壓縮檔是三層的樹（`├─` `└─` `│` Overlay0、隱藏檔不列、滿了最後一列 `…`、`(empty)`、`(permission denied)`）、圖片第一列灰字 `data:image/png · 12.4 KB` 空一列接 data URI（超過 1 MB 只寫一句）、PDF 第一列灰字頁數空一列接前 20 頁的文字（最多 128 KB，沒有文字寫 `(no extractable text)`）、文字檔上色加行號、其他是十六進位一律一列 8 bytes 接 ASCII（33 格，並排、上下疊都一樣）、讀不到 `(unreadable)`；metadata 四列 `TYPE`（`text` `private key` `image` `PDF` `archive` `binary`、`link → <目標>`、`directory`）`SIZE`（1024 進位一位小數、目錄寫 `N items` 不算隱藏檔）`MODIFIED`（本地時間到分）`MODE`（照 `ls -l`），label 層色、值 Text；設定 `hide_secrets_in_preview`（預設關）打開時私鑰露第一行接灰字、其他只有擁有者能讀的檔一行都不露（目錄不算），metadata 照常 | I 的 `Identity file`（假的 `.ssh`）；I `[2] FILES` 的 `[/] Find`（假的專案，每一種檔各一個）；global 的 `Hide secrets in preview` | `TestFilePickerLayout`、`TestFilePickerHiddenAndOrder`、`TestFilePreviewKinds`、`TestFilePreviewHidesSecrets`、`TestFileInfo` |
| file-picker §按鍵 | 清單 `Enter`：目錄進去（清篩選、cursor 在 `..` 下第一項）、檔案選中寫回關掉；目錄永遠不會被選中 | 同上 | `TestFilePicker` |
| file-picker §按鍵 | 打字列：一般字照「比對」篩選目前這一層；路徑（`/`、`~/` 開頭）一直留在打字列、以最後一個 `/` 切成目錄與篩選字；目錄不存在或有 `..` 這一段：清單是空的；`*` `?` wildcard（`**` 往下找）、每列寫相對於起點的路徑、標題寫起點、沒有 `..`；hint 三種 | 同上，打 `~/.ss`、`~/.ssh/id_`、`~/nope/x`、`~/../x`、`~/.ssh/**/*.pem` | `TestFilePicker`、`TestFilePickerPathRules`、`TestGlobMatch`、`TestFilePickerHint` |

## 熱鍵標記（M5）在展示場用到的地方

| 情境 | 展示場的登記 |
|---|---|
| 首字母 | `[m]ark`、`[D]rag`、`[V]iew` |
| 在 label 中間 | `UR[L]`（I `[2] FILES` 的 copy URL） |
| 字母不在 label 裡 | `[x] Delete` |
| 含 modifier | `[Alt-t]erm` |
| 多字元 | `[go]to` |
| 數字 | select 選項的 `[1] clock`、`[2] matrix`；global operation popup 裡的數字熱鍵 |
| core key | `[Enter] Logs` |
| 沒有熱鍵 | `Empty`、`Loading`、`Fail` |
| label 本身寫了鍵 | `[/] Find`、`[/] Search` |

## glyph（C4 的清單）

規範用 code point 指定的 glyph。C4 要每一個在展示場出現一次、截圖看三件事（游標前進幾格、清不清楚、意思對不對得上）。

| code point | 名稱 | 規範 | 展示場 |
|---|---|---|---|
| U+E0BA | `nf-ple-lower_right_triangle` | panel §膠囊串（左端） | 每個 panel 標題、chip 列、popup 的 tab |
| U+E0B0 | `nf-pl-left_hard_divider` | panel §膠囊串（右端、換色接縫）；popup §外框（cursor 尾端的箭頭） | 同上；每個 cursor |
| U+E0B1 | `nf-pl-left_soft_divider` | panel §膠囊串（同色接縫） | P `[1] PODS` 的 tab |
| U+E0B6 | `nf-ple-left_half_circle_thick` | popup §外框（標題膠囊左端） | 每種 popup |
| U+E0B4 | `nf-ple-right_half_circle_thick` | popup §外框（標題膠囊右端） | 每種 popup |
| U+F02D8 | `nf-md-hexagon` | menu §長相、note §key reference（區塊標題） | Space menu；任一處按 `?` |
| U+F160 | `nf-fa-sort_amount_asc` | panel §一般清單（升冪；C3） | P `[1] PODS` 表頭 |
| U+F161 | `nf-fa-sort_amount_desc` | panel §一般清單（降冪；C3） | 同上 |
| U+F105 | `nf-fa-angle_right` | finder §長相、panel §panel filter、file-picker（打字列最前面） | I `[/] Find`；P `[1] PODS` 按 `/` |
| U+F0132 | `nf-md-checkbox_marked` | panel §一般清單（標記）、checkbox、finder §清單的標記 | P 標記的列；I 的 `Agent`、`Namespaces` |
| U+F0131 | `nf-md-checkbox_blank_outline` | checkbox、finder §清單的標記 | I 的 `Agent`、`Namespaces` |
| U+F043E | `nf-md-radiobox_marked` | radio、select | I 的 `Layout`、`Profile` |
| U+F043D | `nf-md-radiobox_blank` | radio、select | 同上 |
| U+F0C9 | `nf-fa-bars` | popup §外框（menu） | Space menu |
| U+F059 | `nf-fa-question_circle` | popup §外框、note §key reference | 任一處按 `?` |
| U+F0ECC | `nf-md-shield_alert` | popup §外框、confirm | D「Confirm」 |
| U+F018D | `nf-md-console` | popup §外框、terminal | T 的 terminal popup |
| U+F024 | `nf-fa-flag` | toast（Info） | `Toast Info` |
| U+F078F | `nf-md-alarm_light` | toast（Warning） | `Toast Warning` |
| U+F0238 | `nf-md-fire` | toast（Error）、note §error popup | `Toast Error`；D「Error popup」 |
| U+F0A9E–U+F0AA5 | `nf-md-circle_slice_1` – `_8` | popup §loading | D「Loading popup」；P `Loading` |

規範直接寫出字元（不是 PUA）的符號，一起看：`┏━┓`（focus 的 panel、popup）、`┌─┐`（失焦的 panel）、`┫ ┣` `┤ ├`（模式名的接頭）、
`━`（畫面 chip 列下的分隔線、開關動畫的線、slider 軌道）、`╌`（U+254C，menu 區塊之間）、`─`（popup 裡的分隔線）、`│`（datetime 的
直分隔線）、`…`（截斷）、`•`（password 一般遮罩、表單上的 8 個點）、`●`（password 解鎖遮罩、slider 的值）、`███`（color 的色塊）、
紅色的 `\n` `\t`（貼上的換行與 Tab）。
