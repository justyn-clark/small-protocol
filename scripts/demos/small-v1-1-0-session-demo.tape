Require git
Require jq
Require small

Output "small-session-frames/"

Set Shell "bash"
Set FontFamily "Menlo"
Set FontSize 22
Set Width 1200
Set Height 800
Set Padding 28
Set Framerate 30
Set TypingSpeed 12ms
Set Theme "Grey-green"
Set MarginFill "#09090b"
Set Margin 20
Set BorderRadius 14
Set WindowBar Rings

Hide
Type "export PATH=$SMALL_DEMO_BIN_DIR:$PATH" Enter
Type "git clone --quiet --no-hardlinks $SMALL_DEMO_SOURCE $SMALL_DEMO_WORK" Enter
Type "cd $SMALL_DEMO_WORK" Enter
Type "export PS1='$ '" Enter
Type "intro(){ clear; printf '\n  SMALL v1.1.0\n  Durable sessions on a real project\n\n'; }" Enter
Type "outro(){ clear; printf '\n  Solo by default. Collaborative by choice.\n  Real state. Ready for the next run.\n\n'; }" Enter
Type "intro" Enter
Show
Sleep 1.5s

Hide
Type "clear" Enter
Show
Type "small version" Enter
Sleep 1.2s

Type "small status --json | jq '.plan.tasks_by_status'" Enter
Sleep 2s

Type "small check --strict" Enter
Sleep 1.5s

Type "small migrate --preview --to 2.0.0 --namespace demo --json > m.json" Enter
Sleep 300ms
Type "jq '{to,mode,tasks:(.tasks|length)}' m.json" Enter
Sleep 2s

Type "D=$(jq -r .expected_input_digest m.json)" Enter
Type "small migrate --apply m.json --expect-state $D --json > a.json" Enter
Sleep 300ms
Type "jq '{event_count,idempotent}' a.json" Enter
Sleep 2s

Type "small mode show --json | jq '{mode,sessions:(.active_sessions|length)}'" Enter
Sleep 2s

Type "small session start --label release-demo --json > s.json" Enter
Sleep 300ms
Type "jq '.session|{session_id,label}' s.json" Enter
Sleep 2s

Type "S=$(jq -r .session.session_id s.json)" Enter
Type "F=$(small mode show --json | jq -r .frontier)" Enter
Type "small mode set collaborative --session $S --expect-state $F --reason demo --json > c.json" Enter
Sleep 300ms
Type "jq '{changed,mode}' c.json" Enter
Sleep 2s

Type "small check --strict" Enter
Sleep 1.5s

Hide
Type "outro" Enter
Show
Sleep 3s
