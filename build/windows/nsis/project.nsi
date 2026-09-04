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
## !define INFO_PROJECTNAME    "my-project" # Default "freehand"
## !define INFO_COMPANYNAME    "My Company" # Default "tnware"
## !define INFO_PRODUCTNAME    "My Product Name" # Default "Freehand"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "0.1.0-alpha.1"
## !define INFO_COPYRIGHT      "(c) Now, My Company" # Default "Copyright 2026 Tyler Woods"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
## !define WAILS_INSTALL_SCOPE     "user"             # Default "machine" - set to "user" for per-user install ($LOCALAPPDATA) without UAC prompt
####
## Keep these policy values outside generated wails_tools.nsh. INFO_PRODUCTVERSION
## remains the human-readable semver; Windows requires a numeric four-part value
## for the executable version resource.
####
!ifndef INFO_BINARYVERSION
    !define INFO_BINARYVERSION "0.1.0.1"
!endif
!ifndef UNINST_KEY_NAME
    !define UNINST_KEY_NAME "io.github.tnware.freehand"
!endif
!ifndef WAILS_INSTALL_SCOPE
    !define WAILS_INSTALL_SCOPE "user"
!endif

## Include the wails tools
####
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_BINARYVERSION}"
VIFileVersion    "${INFO_BINARYVERSION}"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "StrFunc.nsh"
${StrStr}
${UnStrStr}

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
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture
   Call EnsureAppNotRunning
FunctionEnd

Function EnsureAppNotRunning
    checkAgain:
    nsExec::ExecToStack '"$SYSDIR\tasklist.exe" /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH'
    Pop $0
    Pop $1
    ${StrStr} $2 $1 "${PRODUCT_EXECUTABLE}"
    StrCmp $2 "" notRunning

    IfSilent silentRunning interactiveRunning
    interactiveRunning:
        MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION \
            "${INFO_PRODUCTNAME} is running.$\r$\n$\r$\nQuit it from the tray, then choose Retry." \
            IDRETRY checkAgain
        SetErrorLevel 32
        Quit
    silentRunning:
        SetErrorLevel 32
        Quit
    notRunning:
FunctionEnd

Function un.onInit
    nsExec::ExecToStack '"$SYSDIR\tasklist.exe" /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH'
    Pop $0
    Pop $1
    ${UnStrStr} $2 $1 "${PRODUCT_EXECUTABLE}"
    StrCmp $2 "" appNotRunning

    IfSilent silentUninstallRunning interactiveUninstallRunning
    interactiveUninstallRunning:
        MessageBox MB_OK|MB_ICONSTOP \
            "${INFO_PRODUCTNAME} is running.$\r$\n$\r$\nQuit it from the tray before uninstalling."
    silentUninstallRunning:
        SetErrorLevel 32
        Quit
    appNotRunning:
FunctionEnd

Function VerifyWebView2Runtime
    SetRegView 64
    ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
    StrCmp $0 "" checkUserRuntime runtimeReady
    checkUserRuntime:
        !if "${WAILS_INSTALL_SCOPE}" == "user"
            ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
            StrCmp $0 "" runtimeMissing runtimeReady
        !else
            Goto runtimeMissing
        !endif
    runtimeMissing:
        SetErrorLevel 34
        MessageBox MB_OK|MB_ICONSTOP \
            "Microsoft Edge WebView2 Runtime could not be installed.$\r$\n$\r$\nInstall WebView2 Runtime, then run this installer again."
        Abort
    runtimeReady:
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime
    Call VerifyWebView2Runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files
    File /oname=THIRD_PARTY_NOTICES.txt "..\..\..\THIRD_PARTY_NOTICES.md"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    
    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall" 
    !insertmacro wails.setShellContext

    # Runtime settings, Credential Manager entries, and WebView data are user
    # data. Uninstall removes only files installed beneath $INSTDIR.
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 executableRemoved
    ClearErrors
    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    IfErrors uninstallFailed
    executableRemoved:

    Delete "$INSTDIR\THIRD_PARTY_NOTICES.txt"
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
    RMDir "$INSTDIR"
    Goto uninstallComplete

    uninstallFailed:
        SetErrorLevel 33
        MessageBox MB_OK|MB_ICONSTOP \
            "${INFO_PRODUCTNAME} could not be removed.$\r$\n$\r$\nQuit it from the tray and try again."
        Abort
    uninstallComplete:
SectionEnd
