Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
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
## The following information is taken from the wails_tools.nsh file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "my-project" # Default "typewriter"
## !define INFO_COMPANYNAME    "My Company" # Default "My Company"
## !define INFO_PRODUCTNAME    "My Product Name" # Default "My Product"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "0.1.0"
## !define INFO_COPYRIGHT      "(c) Now, My Company" # Default "© 2026, My Company"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
## !define WAILS_INSTALL_SCOPE     "user"             # Default "machine" - set to "user" for per-user install ($LOCALAPPDATA) without UAC prompt
####
## Typewriter installs for the user who runs it, in %LOCALAPPDATA%\Programs\Typewriter, with no admin prompt.
## That way the app can also update itself later (a program in Program Files cannot replace its own file).
!ifndef WAILS_INSTALL_SCOPE
    !define WAILS_INSTALL_SCOPE "user"
!endif
!ifndef REQUEST_EXECUTION_LEVEL
    !define REQUEST_EXECUTION_LEVEL "user"
!endif

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
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.
!define MUI_WELCOMEPAGE_TEXT "This installs ${INFO_PRODUCTNAME} ${INFO_PRODUCTVERSION} for you (no administrator needed).$\r$\n$\r$\nIf ${INFO_PRODUCTNAME} is installed already, it is updated. Your writing is kept: it is in your own data folder, not in the program folder.$\r$\n$\r$\nClose ${INFO_PRODUCTNAME} before you go on."
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}" # "Start Typewriter" on the last page
!define MUI_FINISHPAGE_RUN_TEXT "Start ${INFO_PRODUCTNAME}"

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_COMPONENTS # The program, and a desktop shortcut that you can leave out.
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif
InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation" # an update goes into the folder of the earlier install
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

# The program file cannot be replaced while the app runs: ask to close it first.
!macro typewriter.waitUntilClosed
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 +6
    ClearErrors
    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    IfErrors 0 +3
        MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "${INFO_PRODUCTNAME} is open. Close it, then press Retry." IDRETRY -4
        Abort
!macroend

Section "${INFO_PRODUCTNAME}" SecApp
    SectionIn RO # the program itself cannot be left out
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    !insertmacro typewriter.waitUntilClosed

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
    # where it is installed, so a newer installer (and the app, after it updated itself) can find the entry
    WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
SectionEnd

Section "Desktop shortcut" SecDesktop
    !insertmacro wails.setShellContext
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} "The ${INFO_PRODUCTNAME} program, a Start menu entry, and an uninstaller."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "A shortcut on the desktop."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "uninstall"
    !insertmacro wails.setShellContext
    !insertmacro typewriter.waitUntilClosed
    DetailPrint "Your writing is kept. It is in your data folder (normally %APPDATA%\typewriter)."

    # The WebView2 folder ($AppData\${PRODUCT_EXECUTABLE}) is kept: it holds the choices of this computer (theme, the look of
    # the page, the Vim keys, the place you were writing), and the portable copy of the app uses it too.

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
