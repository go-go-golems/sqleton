Output doc/gifs/demo.mov

Set FontSize 14
Set Width 800
Set Height 600

Set TypingSpeed 0.01

Type "sqleton db ls --fields name,type,hostname" Enter
Sleep 1200ms

Type "sqleton run examples/show-processlist.sql --output yaml" Enter
Sleep 1200ms

Sleep 2000ms