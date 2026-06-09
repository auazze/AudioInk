Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them with the values from ProjectInfo.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the ProjectInfo file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "MyProject" # Default "{{.Name}}"
## !define INFO_COMPANYNAME    "MyCompany" # Default "{{.Info.CompanyName}}"
## !define INFO_PRODUCTNAME    "MyProduct" # Default "{{.Info.ProductName}}"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "{{.Info.ProductVersion}}"
## !define INFO_COPYRIGHT      "Copyright" # Default "{{.Info.Copyright}}"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
####
## Include the wails tools
####
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

# Custom reinstall check page (before Welcome) — uses English buttons via MUI
Page custom reinstallCheckPage reinstallCheckLeave

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Var ReinstallRadio
Var UninstallRadio
Var UninstStr

Function reinstallCheckPage
    SetRegView 64
    ReadRegStr $UninstStr HKLM "${UNINST_KEY}" "UninstallString"
    ${If} $UninstStr == ""
        Abort ; not installed — skip this page
    ${EndIf}

    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 30u "AudioInk is already installed. Choose an action and click Next:"
    Pop $0

    ${NSD_CreateRadioButton} 20u 40u -20u 15u "Reinstall (remove old version, install new)"
    Pop $ReinstallRadio
    ${NSD_SetState} $ReinstallRadio ${BST_CHECKED}

    ${NSD_CreateRadioButton} 20u 60u -20u 15u "Uninstall only"
    Pop $UninstallRadio

    nsDialogs::Show
FunctionEnd

Function reinstallCheckLeave
    ${NSD_GetState} $ReinstallRadio $0
    ${If} $0 == ${BST_CHECKED}
        ; Reinstall: silently remove old, continue install
        ExecWait '$UninstStr /S'
    ${Else}
        ; Uninstall only: run uninstaller with UI, then abort
        ExecWait $UninstStr
        Abort
    ${EndIf}
FunctionEnd

# Macro to register context menu for one audio extension
!macro RegisterContextMenu EXT
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk" "" "AudioInk: Fix name && tags"
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk" "Icon" "$INSTDIR\icon.ico"
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk" "MultiSelectModel" "Player"
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk\command" "" '"$INSTDIR\${PRODUCT_EXECUTABLE}" --fix "%1"'
!macroend

!macro UnregisterContextMenu EXT
    DeleteRegKey HKCR "SystemFileAssociations\${EXT}\shell\AudioInk"
!macroend

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}" # Install directly to Program Files\AudioInk (no extra subfolder).
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture

   # Kill running AudioInk processes before installing
   nsExec::Exec 'taskkill /f /im ${PRODUCT_EXECUTABLE}'
   Pop $R0 # discard taskkill exit code

   # Must match the 64-bit registry view used by wails.writeUninstaller
   SetRegView 64
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    # Clean up legacy nested subfolder from older installs
    RMDir /r "$INSTDIR\AudioInk"

    !insertmacro wails.files

    # Copy icon for context menu (avoids loading 14MB exe for icon extraction)
    File "/oname=icon.ico" "..\icon.ico"

    # Bundle the audio toolchain next to AudioInk.exe. The app resolves these
    # by absolute path (dir of os.Executable()), so they MUST sit in $INSTDIR
    # — context-menu launches run from an arbitrary CWD, not from here.
    #   ffmpeg/ffprobe → transcode, loudness, silence, health, repair
    #   fpcalc         → local duplicate detection (Chromaprint)
    File "/oname=ffmpeg.exe"  "..\..\bin\ffmpeg.exe"
    File "/oname=ffprobe.exe" "..\..\bin\ffprobe.exe"
    File "/oname=fpcalc.exe"  "..\..\bin\fpcalc.exe"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    # Register context menu for audio files
    !insertmacro RegisterContextMenu ".mp3"
    !insertmacro RegisterContextMenu ".flac"
    !insertmacro RegisterContextMenu ".ogg"
    !insertmacro RegisterContextMenu ".m4a"
    !insertmacro RegisterContextMenu ".wav"
    !insertmacro RegisterContextMenu ".wma"
    !insertmacro RegisterContextMenu ".opus"

    !insertmacro wails.writeUninstaller

    # Store install location so reinstall/uninstall detection works from any path
    SetRegView 64
    WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    # Remove context menu for audio files
    !insertmacro UnregisterContextMenu ".mp3"
    !insertmacro UnregisterContextMenu ".flac"
    !insertmacro UnregisterContextMenu ".ogg"
    !insertmacro UnregisterContextMenu ".m4a"
    !insertmacro UnregisterContextMenu ".wav"
    !insertmacro UnregisterContextMenu ".wma"
    !insertmacro UnregisterContextMenu ".opus"

    !insertmacro wails.deleteUninstaller
SectionEnd
