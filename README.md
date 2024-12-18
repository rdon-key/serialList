# How to use

Windowsマシンで、コマンドラインからbin/serialList.exeを実行してください。
シリアルポートの状況が表示され、USB接続のものはVIDとPIDが表示されます。
VID:2E8A のものがRaspberryPiPicoである可能性が高いです。

実行例です。この例ではCOM5がRaspberryPiPicoに接続されています。

```
COM1 : ready
COM5 [VID:2E8A Raspberry Pi Foundation PID:0003] : ready
```

# How to build 

src/serialList.go　を次のコマンドで実行してください。

```
go run src/serialList.go
```

go buildでexeファイルを作成することもできます。
