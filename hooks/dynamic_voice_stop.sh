#!/usr/bin/env bash
# ========================================
# Claude Code Stop Hook - 音声読み上げスクリプト
# ========================================
# 
# 概要: Claude Codeの Stop フックで呼び出され、最後のアシスタント回答を
#       VOICEVOX または AivisSpeech を使って音声で読み上げるスクリプト
#
# 依存関係:
#   - VOICEVOX ENGINE (http://127.0.0.1:50021) または
#   - AivisSpeech (http://127.0.0.1:10101)
#   - jq (JSON解析)
#   - afplay (macOS音声再生)
#   - curl (HTTP通信)
#
# 設定可能項目:
#   - UUID_MODE: UUID検索モード (true=全コンテンツ/false=軽量)
#   - READING_MODE: 読み上げ範囲 (first_line/after_first/full_text/char_limit)
#   - MAX_CHARS: 文字数制限値
#   - ENGINE_PRIORITY: エンジン優先順位 (voicevox/aivisspeech)
#   - VOICEVOX_SPEAKER: VOICEVOXの話者ID
#   - AIVISSPEECH_SPEAKER: AivisSpeechの話者ID
#
# 使用方法:
#   - Claude Code の Stop フックとして自動実行
#   - 手動設定変更: bash voice_config.sh
#
# ログ出力: /tmp/voicestop_debug.log
# ========================================

# スクリプト実行設定
set -e                        # エラー発生時に即座に終了
read -r json || json="{}"     # stdin からJSONを読み取り（失敗時はデフォルト値）

# 音声エンジン設定
#ENGINE_PRIORITY="voicevox"    # エンジン優先順位（voicevox/aivisspeech）
ENGINE_PRIORITY="aivisspeech"    # エンジン優先順位（voicevox/aivisspeech）
#VOICEVOX_SPEAKER=47          # VOICEVOX話者ID（ナースロボ＿タイプT）
VOICEVOX_SPEAKER=3
#AIVISSPEECH_SPEAKER=888753760  # AivisSpeech話者ID（Anneli ノーマル）
AIVISSPEECH_SPEAKER=1512153248

# 利用可能なAivisSpeech話者ID:
# 888753760=Anneli ノーマル, 888753761=Anneli 通常, 888753762=Anneli テンション高め
# 888753763=Anneli 落ち着き, 888753764=Anneli 上機嫌, 888753765=Anneli 怒り・悲しみ
# 606865152=fumifumi, 933744512=peach, 706073888=white
# 1431611904=まい, 376644064=桜音, 1325133120=花音

# ========================================
# 設定変更コマンド処理（オプション機能）
# ========================================
# 
# 受信JSONに設定変更コマンドが含まれている場合、
# スクリプト自体の設定値を動的に変更する
#
# 対応コマンド:
#   uuid_on/uuid_off: UUID検索モードの切り替え
#   mode_first/mode_after/mode_full/mode_char: 読み上げモード変更
#   show_config: 現在の設定表示
#
# コマンドライン引数での設定変更チェック（スクリプト単体実行時）
if [[ $# -gt 0 ]]; then
    case "$1" in
        "config")
            echo "🎵 音声読み上げ設定変更"
            echo ""
            echo "📊 現在の設定:"
            grep -E "^(UUID_MODE|READING_MODE|MAX_CHARS|MAX_LINES)=" "$0"
            echo ""
            echo "変更したい項目を選択してください:"
            echo "1) UUID検索 ON/OFF"
            echo "2) 読み上げモード変更"
            echo "3) 行数制限値変更"
            echo "4) 文字数制限値変更"
            echo "5) 音声エンジン優先順位変更"
            echo "6) 設定確認のみ"
            echo "7) 終了"
            read -p "選択 (1-7): " choice
            
            case $choice in
                1)
                    current_uuid=$(grep '^UUID_MODE=' "$0" | cut -d'=' -f2)
                    if [[ "$current_uuid" == "true" ]]; then
                        sed -i '' 's/^UUID_MODE=.*/UUID_MODE=false/' "$0"
                        echo "✅ UUID検索モード: OFF"
                    else
                        sed -i '' 's/^UUID_MODE=.*/UUID_MODE=true/' "$0"
                        echo "✅ UUID検索モード: ON"
                    fi
                    ;;
                2)
                    echo "読み上げモードを選択:"
                    echo "a) first_line: 最初の改行まで"
                    echo "b) line_limit: 指定行数まで"
                    echo "c) after_first: 改行後の内容"
                    echo "d) full_text: 全文読み上げ"
                    echo "e) char_limit: 文字数制限"
                    read -p "選択 (a-e): " mode_choice
                    case $mode_choice in
                        a) sed -i '' 's/^READING_MODE=.*/READING_MODE="first_line"/' "$0"; echo "✅ 最初の改行まで" ;;
                        b) sed -i '' 's/^READING_MODE=.*/READING_MODE="line_limit"/' "$0"; echo "✅ 行数制限" ;;
                        c) sed -i '' 's/^READING_MODE=.*/READING_MODE="after_first"/' "$0"; echo "✅ 改行後の内容" ;;
                        d) sed -i '' 's/^READING_MODE=.*/READING_MODE="full_text"/' "$0"; echo "✅ 全文読み上げ" ;;
                        e) sed -i '' 's/^READING_MODE=.*/READING_MODE="char_limit"/' "$0"; echo "✅ 文字数制限" ;;
                        *) echo "❌ 無効な選択" ;;
                    esac
                    ;;
                3)
                    current_lines=$(grep '^MAX_LINES=' "$0" | cut -d'=' -f2)
                    read -p "行数制限値を入力 (現在: $current_lines): " new_lines
                    if [[ "$new_lines" =~ ^[0-9]+$ ]] && [ "$new_lines" -gt 0 ]; then
                        sed -i '' "s/^MAX_LINES=.*/MAX_LINES=$new_lines/" "$0"
                        echo "✅ 行数制限: ${new_lines}行"
                    else
                        echo "❌ 無効な行数"
                    fi
                    ;;
                4)
                    current_chars=$(grep '^MAX_CHARS=' "$0" | cut -d'=' -f2)
                    read -p "文字数制限値を入力 (現在: $current_chars): " new_chars
                    if [[ "$new_chars" =~ ^[0-9]+$ ]] && [ "$new_chars" -gt 0 ]; then
                        sed -i '' "s/^MAX_CHARS=.*/MAX_CHARS=$new_chars/" "$0"
                        echo "✅ 文字数制限: ${new_chars}文字"
                    else
                        echo "❌ 無効な文字数"
                    fi
                    ;;
                5)
                    echo "音声エンジン優先順位を選択:"
                    echo "a) VOICEVOX優先 (推奨)"
                    echo "b) AivisSpeech優先"
                    read -p "選択 (a-b): " engine_choice
                    case $engine_choice in
                        a) sed -i '' 's/^ENGINE_PRIORITY=.*/ENGINE_PRIORITY="voicevox"/' "$0"; echo "✅ VOICEVOX優先" ;;
                        b) sed -i '' 's/^ENGINE_PRIORITY=.*/ENGINE_PRIORITY="aivisspeech"/' "$0"; echo "✅ AivisSpeech優先" ;;
                        *) echo "❌ 無効な選択" ;;
                    esac
                    ;;
                6)
                    echo "📊 詳細設定:"
                    grep -E "^(UUID_MODE|READING_MODE|MAX_CHARS|MAX_LINES|ENGINE_PRIORITY)=" "$0"
                    ;;
                7)
                    echo "👋 終了"
                    ;;
                *)
                    echo "❌ 無効な選択"
                    ;;
            esac
            exit 0
            ;;
    esac
fi

config_command=$(echo "$json" | jq -r '.config_command // empty' 2>/dev/null || echo "")
if [[ -n "$config_command" ]]; then
    case "$config_command" in
        "uuid_on") 
            # UUID検索モードを有効化（全コンテンツ取得・重い処理）
            sed -i '' 's/^UUID_MODE=.*/UUID_MODE=true/' "$0"
            echo "✅ UUID検索モード: ON" >&2
            exit 0 ;;
        "uuid_off") 
            # UUID検索モードを無効化（軽量・高速処理）
            sed -i '' 's/^UUID_MODE=.*/UUID_MODE=false/' "$0"
            echo "✅ UUID検索モード: OFF" >&2
            exit 0 ;;
        "mode_first") 
            # 読み上げモード: 最初の改行まで
            sed -i '' 's/^READING_MODE=.*/READING_MODE="first_line"/' "$0"
            echo "✅ 読み上げモード: 最初の改行まで" >&2
            exit 0 ;;
        "mode_after") 
            # 読み上げモード: 改行後の内容のみ
            sed -i '' 's/^READING_MODE=.*/READING_MODE="after_first"/' "$0"
            echo "✅ 読み上げモード: 改行後の内容" >&2
            exit 0 ;;
        "mode_full") 
            # 読み上げモード: 全文読み上げ
            sed -i '' 's/^READING_MODE=.*/READING_MODE="full_text"/' "$0"
            echo "✅ 読み上げモード: 全文読み上げ" >&2
            exit 0 ;;
        "mode_line") 
            # 読み上げモード: 行数制限
            sed -i '' 's/^READING_MODE=.*/READING_MODE="line_limit"/' "$0"
            echo "✅ 読み上げモード: 行数制限" >&2
            exit 0 ;;
        "mode_char") 
            # 読み上げモード: 文字数制限
            sed -i '' 's/^READING_MODE=.*/READING_MODE="char_limit"/' "$0"
            echo "✅ 読み上げモード: 文字数制限" >&2
            exit 0 ;;
        "engine_voicevox")
            # 音声エンジンをVOICEVOXに設定
            sed -i '' 's/^ENGINE_PRIORITY=.*/ENGINE_PRIORITY="voicevox"/' "$0"
            echo "✅ 音声エンジン優先順位: VOICEVOX" >&2
            exit 0 ;;
        "engine_aivisspeech")
            # 音声エンジンをAivisSpeechに設定
            sed -i '' 's/^ENGINE_PRIORITY=.*/ENGINE_PRIORITY="aivisspeech"/' "$0"
            echo "✅ 音声エンジン優先順位: AivisSpeech" >&2
            exit 0 ;;
        "show_config")
            # 現在の設定を表示
            echo "📊 現在の設定:" >&2
            grep -E "^(UUID_MODE|READING_MODE|MAX_CHARS|MAX_LINES|ENGINE_PRIORITY)=" "$0" >&2
            exit 0 ;;
    esac
fi

# ========================================
# デバッグログ出力
# ========================================
echo "🔧 voicestop.sh が呼び出されました $(date)" >> /tmp/voicestop_debug.log
echo "🔧 受信JSON: $json" >> /tmp/voicestop_debug.log

# ========================================
# ① transcript ファイルの取得・検証
# ========================================
# 
# Claude Code の Stop フックから transcript_path を取得
# 提供されない場合は最新のセッションファイルを自動検索
#

# JSONから transcript_path を抽出
path=$(jq -r '.transcript_path // empty' <<<"$json" 2>/dev/null || echo "")

# transcript_path が提供されない場合は最新ファイルを検索
if [[ -z "$path" ]] || [[ "$path" == "null" ]]; then
    echo "🔧 transcript_pathが提供されないため、最新ファイルを検索します" >> /tmp/voicestop_debug.log
    # ~/.claude/projects から最新の .jsonl ファイルを検索
    path=$(find ~/.claude/projects -name "*.jsonl" -type f -exec ls -t {} + 2>/dev/null | head -1)
    echo "🔧 検出したファイル: $path" >> /tmp/voicestop_debug.log
fi

# transcript ファイルの存在確認
if [[ -z "$path" ]] || [[ ! -f "$path" ]]; then
    echo "⚠️ transcript file not found or empty: '$path'" >&2
    echo "🔧 ファイルが見つかりません: $path" >> /tmp/voicestop_debug.log
    exit 0  # 正常終了（ファイルが見つからない場合もエラーとしない）
fi

# ========================================
# ② アシスタントメッセージのテキスト抽出
# ========================================
# 
# 2つのモードでテキスト取得を行う:
# 1. UUID検索モード: 完全なメッセージ取得（重い・正確）
# 2. 軽量モード: 最初のテキストのみ取得（軽い・高速）
#

if [ "$UUID_MODE" = true ]; then
    # ========================================
    # UUID検索モード（全コンテンツ取得・重い処理）
    # ========================================
    # 
    # 最新のassistantメッセージのUUIDを特定し、
    # そのUUIDに関連する全てのレコードからテキストを抽出・結合
    #
    echo "🔧 UUID検索モード使用" >> /tmp/voicestop_debug.log
    
    if command -v tac >/dev/null 2>&1; then
        # Linux: tac コマンドでファイルを逆順読み込み
        latest_uuid=$(tac "$path" | jq -r 'select(.type == "assistant") | .uuid' 2>/dev/null | head -n 1)
        text=$(tac "$path" | jq -r --arg uuid "$latest_uuid" 'select(.uuid == $uuid and .message.role == "assistant") | .message.content[]? | select(.type == "text") | .text' 2>/dev/null | paste -sd ' ' -)
    else
        # macOS: tail -r コマンドでファイルを逆順読み込み  
        latest_uuid=$(tail -r "$path" | jq -r 'select(.type == "assistant") | .uuid' 2>/dev/null | head -n 1)
        text=$(tail -r "$path" | jq -r --arg uuid "$latest_uuid" 'select(.uuid == $uuid and .message.role == "assistant") | .message.content[]? | select(.type == "text") | .text' 2>/dev/null | paste -sd ' ' -)
    fi
    echo "🔧 検出したUUID: $latest_uuid" >> /tmp/voicestop_debug.log
else
    # ========================================
    # 軽量モード（最初のtextのみ・高速）
    # ========================================
    # 
    # 最新のassistantメッセージの最初のtextコンテンツのみ取得
    # 処理速度重視、複数のテキストブロックがあっても最初の1つのみ
    #
    echo "🔧 軽量モード使用" >> /tmp/voicestop_debug.log
    
    if command -v tac >/dev/null 2>&1; then
        # Linux: tac + jq + head で最初のテキストを取得
        text=$(tac "$path" | jq -r 'select(.type == "assistant") | .message.content[]? | select(.type == "text") | .text' 2>/dev/null | head -n 1)
    else
        # macOS: tail -r + jq + head で最初のテキストを取得
        text=$(tail -r "$path" | jq -r 'select(.type == "assistant") | .message.content[]? | select(.type == "text") | .text' 2>/dev/null | head -n 1)
    fi
fi

echo "🔧 取得したテキスト長: ${#text}" >> /tmp/voicestop_debug.log

# ========================================
# テキスト取得結果の検証
# ========================================
if [[ -z "$text" || "$text" == "null" ]]; then
    echo "⚠️ No assistant message found in transcript" >&2
    exit 0  # 非ブロッキング終了（エラーとして扱わない）
fi

# ========================================
# ③ 読み上げ設定値（ユーザーカスタマイズ可能）
# ========================================
UUID_MODE=false            # UUID検索モード（false=軽量、true=全コンテンツ取得）
READING_MODE="first_line"  # 読み上げモード（後述の5種類から選択）
MAX_CHARS=500              # 文字数制限モード時の最大文字数
MAX_LINES=3                # 行数制限モード時の最大行数

# ========================================
# 読み上げモード詳細説明
# ========================================
# first_line:  最初の改行まで（初期値・高速・推奨）
#              - 長い回答の概要のみ読み上げ
#              - 処理速度最優先
#
# line_limit:  指定行数まで読み上げ（NEW!）
#              - MAX_LINES 設定値の行数まで読み上げ
#              - 概要＋詳細の適度なバランス
#
# after_first: 改行後の内容のみ
#              - 概要行をスキップして詳細部分を読み上げ
#              - 説明文の本文のみ聞きたい場合
#
# full_text:   全文読み上げ（改行をスペースに変換）
#              - 完全な内容を全て読み上げ
#              - 時間がかかるが情報は完全
#
# char_limit:  指定文字数で切断
#              - MAX_CHARS 設定値まで読み上げ
#              - 時間制御したい場合

# ========================================
# ④ 読み上げ範囲の処理実行
# ========================================
case "$READING_MODE" in
    "first_line")
        # 最初の改行まで（初期値・高速）
        text=$(echo "$text" | head -n 1)
        echo "🔧 読み上げモード: 最初の改行まで" >> /tmp/voicestop_debug.log
        ;;
    "line_limit")
        # 指定行数まで読み上げ（head -n N で N行まで取得）
        text=$(echo "$text" | head -n $MAX_LINES | tr '\n' ' ')
        echo "🔧 読み上げモード: ${MAX_LINES}行制限" >> /tmp/voicestop_debug.log
        ;;
    "after_first")
        # 改行後の内容のみ（tail -n +2 で2行目以降を取得）
        text=$(echo "$text" | tail -n +2 | tr '\n' ' ')
        echo "🔧 読み上げモード: 改行後の内容" >> /tmp/voicestop_debug.log
        ;;
    "full_text")
        # 全文読み上げ（改行をスペースに変換して1行にまとめる）
        text=$(echo "$text" | tr '\n' ' ')
        echo "🔧 読み上げモード: 全文読み上げ" >> /tmp/voicestop_debug.log
        ;;
    "char_limit")
        # 指定文字数で切断（cut -c1-N で N文字まで取得）
        text=$(echo "$text" | tr '\n' ' ' | cut -c1-$MAX_CHARS)
        echo "🔧 読み上げモード: ${MAX_CHARS}文字制限" >> /tmp/voicestop_debug.log
        ;;
    *)
        # 無効なモード指定時のフォールバック（first_line と同じ処理）
        text=$(echo "$text" | head -n 1)
        echo "🔧 読み上げモード: デフォルト（最初の改行まで）" >> /tmp/voicestop_debug.log
        ;;
esac

# 処理結果をログに記録
echo "🔧 処理後テキスト長: ${#text}文字" >> /tmp/voicestop_debug.log

# ========================================
# Markdownフォーマットの除去（mdstrip使用）
# ========================================
# mdstripコマンドが利用可能な場合、Markdown記法を除去
if command -v mdstrip >/dev/null 2>&1; then
    echo "🔧 mdstripでMarkdown記法を除去" >> /tmp/voicestop_debug.log
    # mdstripにテキストを渡して処理
    text=$(echo "$text" | mdstrip)
    echo "🔧 mdstrip処理後テキスト長: ${#text}文字" >> /tmp/voicestop_debug.log
else
    echo "🔧 mdstripが見つからないため、Markdown記法はそのまま" >> /tmp/voicestop_debug.log
fi

# 最終的な読み上げテキストを表示（デバッグ用）
echo "📢 読み上げテキスト: $text" >&2

# ========================================
# ⑤ 音声エンジン検出・接続確認
# ========================================
# 
# 利用可能な音声エンジンを検出し、優先順位に従って使用するエンジンを決定
# VOICEVOX ENGINE (port 50021) または AivisSpeech (port 10101)
#

# 音声エンジンの検出
voicevox_available=false
aivisspeech_available=false

if curl -s "http://127.0.0.1:50021/version" >/dev/null 2>&1; then
    voicevox_available=true
    echo "🔧 VOICEVOX ENGINE が利用可能です" >> /tmp/voicestop_debug.log
fi

if curl -s "http://127.0.0.1:10101/speakers" >/dev/null 2>&1; then
    aivisspeech_available=true
    echo "🔧 AivisSpeech が利用可能です" >> /tmp/voicestop_debug.log
fi

# 使用するエンジンの決定
selected_engine=""
if [[ "$ENGINE_PRIORITY" == "voicevox" ]]; then
    if $voicevox_available; then
        selected_engine="voicevox"
    elif $aivisspeech_available; then
        selected_engine="aivisspeech"
    fi
else
    if $aivisspeech_available; then
        selected_engine="aivisspeech"
    elif $voicevox_available; then
        selected_engine="voicevox"
    fi
fi

if [[ -z "$selected_engine" ]]; then
    echo "⚠️ 利用可能な音声エンジンが見つかりません" >&2
    echo "VOICEVOX ENGINE (http://127.0.0.1:50021) または AivisSpeech (http://127.0.0.1:10101) を起動してください" >&2
    exit 0
fi

echo "🔧 使用する音声エンジン: $selected_engine" >> /tmp/voicestop_debug.log

# ========================================
# ⑥ 音声合成実行
# ========================================
# 
# 選択されたエンジンに応じて音声合成を実行
#

wave_file=$(mktemp).wav

if [[ "$selected_engine" == "voicevox" ]]; then
    # VOICEVOX ENGINE を使用した音声合成
    echo "🔧 VOICEVOX ENGINEで音声合成を実行" >> /tmp/voicestop_debug.log
    
    # 音声クエリ生成
    query_json=$(mktemp)
    if ! curl -s -X POST "http://127.0.0.1:50021/audio_query?speaker=$VOICEVOX_SPEAKER" \
         --get --data-urlencode "text=$text" -o "$query_json"; then
        echo "⚠️ VOICEVOX ENGINE へのクエリ生成に失敗しました" >&2
        rm -f "$query_json"
        exit 0
    fi
    
    # 音声合成
    if ! curl -s -H 'Content-Type: application/json' \
         -d @"$query_json" \
         "http://127.0.0.1:50021/synthesis?speaker=$VOICEVOX_SPEAKER" -o "$wave_file"; then
        echo "⚠️ VOICEVOX ENGINE での音声合成に失敗しました" >&2
        rm -f "$query_json" "$wave_file"
        exit 0
    fi
    
    rm -f "$query_json"
    
elif [[ "$selected_engine" == "aivisspeech" ]]; then
    # AivisSpeech を使用した音声合成
    echo "🔧 AivisSpeechで音声合成を実行" >> /tmp/voicestop_debug.log
    
    # 利用可能な話者を確認
    echo "🔧 AivisSpeech 話者ID: $AIVISSPEECH_SPEAKER" >> /tmp/voicestop_debug.log
    
    # 話者リストをログに出力（デバッグ用）
    available_speakers=$(curl -s "http://127.0.0.1:10101/speakers" 2>/dev/null)
    if [[ -n "$available_speakers" ]]; then
        echo "🔧 利用可能な話者: $available_speakers" >> /tmp/voicestop_debug.log
    fi
    
    # AivisSpeech API呼び出し（VOICEVOX互換形式）
    query_json=$(mktemp)
    if ! curl -s -X POST "http://127.0.0.1:10101/audio_query?speaker=$AIVISSPEECH_SPEAKER" \
         --get --data-urlencode "text=$text" -o "$query_json"; then
        echo "⚠️ AivisSpeech へのクエリ生成に失敗しました" >&2
        rm -f "$query_json"
        exit 0
    fi
    
    # 音声合成
    if ! curl -s -H 'Content-Type: application/json' \
         -d @"$query_json" \
         "http://127.0.0.1:10101/synthesis?speaker=$AIVISSPEECH_SPEAKER" -o "$wave_file"; then
        echo "⚠️ AivisSpeech での音声合成に失敗しました" >&2
        rm -f "$query_json" "$wave_file"
        exit 0
    fi
    
    rm -f "$query_json"
fi

# ========================================
# ⑦ 音声ファイル再生
# ========================================
# 
# 生成された音声ファイルをmacOS標準のafplayコマンドで再生
# バックグラウンド実行で非同期再生、一定時間後に一時ファイルを自動削除
#
if command -v afplay >/dev/null 2>&1; then
    echo "🔊 音声を再生中... (Engine: $selected_engine)" >&2
    afplay "$wave_file" &  # バックグラウンドで再生開始
    
    # 10秒後に一時ファイルを削除（バックグラウンドで実行）
    (sleep 10; rm -f "$wave_file") &
else
    # afplayコマンドが見つからない場合（通常はmacOSなら存在する）
    echo "⚠️ afplayコマンドが見つかりません（macOS標準のはずですが...）" >&2
    rm -f "$wave_file"  # 一時ファイル削除
    exit 0  # 非ブロッキング終了
fi

echo "✅ 音声読み上げ完了" >&2 
