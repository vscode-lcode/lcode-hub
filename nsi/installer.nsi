Name "lcode-hub"
OutFile "lcode-hub-setup.exe"
InstallDir "$PROGRAMFILES\lcode-hub"
RequestExecutionLevel admin

Section "Install"
  SetOutPath "$INSTDIR"

  ; 拷贝主程序
  File "lcode-hub.exe"

  ; 拷贝 nssm.exe
  File "nssm.exe"

  ; 创建卸载程序
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; 注册服务
  ExecWait '"$INSTDIR\nssm.exe" install lcode-hub "$INSTDIR\lcode-hub.exe"'
  ExecWait '"$INSTDIR\nssm.exe" set lcode-hub Start SERVICE_AUTO_START'

  ; 异步启动服务，避免卡住安装器
  Exec '"$INSTDIR\nssm.exe" start lcode-hub'

  ; 卸载信息
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "DisplayName" "lcode-hub"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "Publisher" "shynome"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "DisplayVersion" "2.0.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub" "DisplayIcon" "$INSTDIR\lcode-hub.exe"
SectionEnd

Section "Uninstall"
  ; 停止并删除服务
  Exec '"$INSTDIR\nssm.exe" stop lcode-hub'
  Exec '"$INSTDIR\nssm.exe" remove lcode-hub confirm'

  ; 删除程序文件
  Delete "$INSTDIR\lcode-hub.exe"
  Delete "$INSTDIR\nssm.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  ; 删除卸载注册表信息
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\lcode-hub"
SectionEnd
