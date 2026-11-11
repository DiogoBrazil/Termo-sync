; Run scripts/check-release.py before setting this define. Never set it for an unverified bundle.
!ifndef REDISTRIBUTION_VERIFIED
 !error "Execute scripts/check-release.py e verifique o build FFmpeg antes de empacotar."
!endif
Unicode true
!include "MUI2.nsh"
!include "x64.nsh"
Name "Depoimento Studio"
OutFile "../build/DepoimentoStudio-Setup.exe"
InstallDir "$LOCALAPPDATA\Programs\Depoimento Studio"
RequestExecutionLevel user
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "PortugueseBR"
Function .onInit
 ${IfNot} ${RunningX64}
  MessageBox MB_ICONSTOP "Este aplicativo requer Windows x64."
  Abort
 ${EndIf}
 SetRegView 64
 ReadRegStr $0 HKLM "SOFTWARE\Microsoft\Windows NT\CurrentVersion" "CurrentBuildNumber"
 ${If} $0 < 22000
  MessageBox MB_ICONSTOP "Este aplicativo requer Windows 11 x64."
  Abort
 ${EndIf}
 SetRegView 32
 ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
 ReadRegStr $1 HKLM "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
 ${If} $0 == "0.0.0.0"
  StrCpy $0 ""
 ${EndIf}
 ${If} $1 == "0.0.0.0"
  StrCpy $1 ""
 ${EndIf}
 ${If} $0 == ""
 ${AndIf} $1 == ""
  MessageBox MB_YESNO "Instale o Microsoft Edge WebView2 Evergreen Runtime e execute este instalador novamente. Abrir o site oficial?" IDNO NoWebView
  ExecShell "open" "https://developer.microsoft.com/microsoft-edge/webview2/"
  NoWebView:
  Abort
 ${EndIf}
FunctionEnd
Section "Aplicativo"
 SetShellVarContext current
 SetOutPath "$INSTDIR"
 File "../build/DepoimentoStudio.exe"
 ; Refuse packaging unless the FFmpeg bundle and redistribution evidence were prepared.
 File /r "../build/tools"
 File /r "../build/licenses"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 CreateShortcut "$SMPROGRAMS\Depoimento Studio.lnk" "$INSTDIR\DepoimentoStudio.exe"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio" "DisplayName" "Depoimento Studio"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio" "UninstallString" '"$INSTDIR\Uninstall.exe"'
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio" "InstallLocation" "$INSTDIR"
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio" "NoModify" 1
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio" "NoRepair" 1
SectionEnd
Section "Uninstall"
 SetShellVarContext current
 Delete "$SMPROGRAMS\Depoimento Studio.lnk"
 Delete "$INSTDIR\DepoimentoStudio.exe"
 RMDir /r "$INSTDIR\tools"
 RMDir /r "$INSTDIR\licenses"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\DepoimentoStudio"
 ; API keys remain in Credential Manager unless removed in Settings before uninstall.
SectionEnd
