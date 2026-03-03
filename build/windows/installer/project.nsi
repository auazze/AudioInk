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

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

# Macro to register context menu for one audio extension
!macro RegisterContextMenu EXT
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk" "" "AudioInk: Fix name && tags"
    WriteRegStr HKCR "SystemFileAssociations\${EXT}\shell\AudioInk" "Icon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
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

   # Must match the 64-bit registry view used by wails.writeUninstaller
   SetRegView 64

   # Check if AudioInk is already installed
   ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
   StrCmp $0 "" done_check

   MessageBox MB_YESNOCANCEL|MB_ICONQUESTION \
       "AudioInk is already installed.$\n$\nYes = Reinstall (remove old, install new)$\nNo = Uninstall only$\nCancel = Abort" \
       IDYES do_reinstall IDNO do_uninstall

   # Cancel
   Abort

   do_uninstall:
       # Run uninstaller and wait, then abort installer
       ExecWait $0
       Abort

   do_reinstall:
       # Run uninstaller silently, then continue with install
       ExecWait '$0 /S'

   done_check:
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    # Clean up legacy nested subfolder from older installs
    RMDir /r "$INSTDIR\AudioInk"

    !insertmacro wails.files

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
