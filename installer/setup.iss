; Inno Setup script for the installer. build.ps1 passes in the version.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{6F3B2C1E-8A4D-4E7B-9C2F-5D1A0B7E3F42}
AppName=PC Resource Monitor
AppVersion={#AppVersion}
AppPublisher=glimmercharger
AppPublisherURL=https://github.com/glimmercharger/pc-resource-monitor
DefaultDirName={autopf}\PC Resource Monitor
DefaultGroupName=PC Resource Monitor
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
ChangesEnvironment=yes
LicenseFile=..\LICENSE
SetupIconFile=..\winres\icon.ico
UninstallDisplayIcon={app}\icon.ico
UninstallDisplayName=PC Resource Monitor
WizardStyle=modern
Compression=lzma2/max
SolidCompression=yes
OutputDir=..\dist
OutputBaseFilename=resourcemonitor-setup

[Tasks]
Name: "addtopath"; Description: "Add the ""resourcemonitor"" command to PATH so it works in any terminal"
Name: "desktopicon"; Description: "Create a desktop shortcut"; Flags: unchecked

[Files]
Source: "..\dist\resourcemonitor.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\winres\icon.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\PC Resource Monitor"; Filename: "{app}\resourcemonitor.exe"; IconFilename: "{app}\icon.ico"
Name: "{autodesktop}\PC Resource Monitor"; Filename: "{app}\resourcemonitor.exe"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NotInPath(ExpandConstant('{app}'))

[Run]
Filename: "{app}\resourcemonitor.exe"; Description: "Open PC Resource Monitor"; Flags: nowait postinstall skipifsilent

[Code]
function NotInPath(Dir: string): Boolean;
var
  Path: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    Path := '';
  Result := Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(Path) + ';') = 0;
end;

// take our folder back out of PATH on uninstall
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path, Dir: string;
begin
  if CurUninstallStep <> usPostUninstall then
    exit;
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    exit;
  Dir := ExpandConstant('{app}');
  Path := ';' + Path + ';';
  StringChangeEx(Path, ';' + Dir + ';', ';', True);
  if (Length(Path) > 0) and (Path[1] = ';') then
    Delete(Path, 1, 1);
  if (Length(Path) > 0) and (Path[Length(Path)] = ';') then
    Delete(Path, Length(Path), 1);
  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path);
end;
