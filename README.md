# Serial Port Lister

## 使い方

1. Windowsマシンで、コマンドラインから`bin/serialList.exe`を実行してください。
2. シリアルポートの状況が表示され、USB接続のデバイスについてはVIDとPIDも表示されます。
3. VID:2E8A が表示されている場合、そのポートはRaspberry Pi Picoである可能性が高いです。

### 実行例
以下の例では、COM5がRaspberry Pi Picoに接続されています：

```
COM1 : ready
COM5 [VID:2E8A Raspberry Pi Foundation PID:0003] : ready
```

## ビルド方法

### 直接実行
```
go run src/serialList.go
```

### 実行ファイルの作成
```
go build -o bin/serialList.exe src/serialList.go
```
