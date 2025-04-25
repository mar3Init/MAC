$Env:GOARCH = "386"
C:\Go_old\bin\go.exe  build -o main_32.exe .\main.go .\copy.go .\module.go .\target.go .\search.go .\zip.go .\send.go
$Env:GOARCH = "amd64"
C:\Go_old\bin\go.exe  build -o main_64.exe .\main.go .\copy.go .\module.go .\target.go .\search.go .\zip.go .\send.go

& 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe'  sign /f .\\cert.pfx /p 0517 /fd SHA256  .\\main_64.exe
& 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe'  sign /f .\\cert.pfx /p 0517 /fd SHA256  .\\main_32.exe