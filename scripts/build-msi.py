#!/usr/bin/env python3
"""Build the all-users Windows installer (.msi) from build/portable with wixl (msitools).

Releases stay behind scripts/check-release.py. --teste-local skips that check and marks the file
name, for trying the installation on the developer's own machine only.
"""
import argparse
import hashlib
import json
import os
import subprocess
import sys
from pathlib import Path
from xml.sax.saxutils import quoteattr

root = Path(__file__).resolve().parent.parent
parser = argparse.ArgumentParser()
parser.add_argument('--teste-local', action='store_true', help='pula a verificação de redistribuição do FFmpeg')
args = parser.parse_args()

if not args.teste_local:
    if subprocess.run([sys.executable, str(root / 'scripts/check-release.py')]).returncode:
        raise SystemExit('Instalador de release não gerado. Para testar na própria máquina, use --teste-local.')

version = json.loads((root / 'packaging/windows/info.json').read_text())['fixed']['file_version']
portable = root / 'build/portable'
exe = portable / 'DepoimentoStudio.exe'
if not exe.is_file() or exe.read_bytes() != (root / 'build/DepoimentoStudio.exe').read_bytes():
    raise SystemExit('Atualize build/portable com o executável compilado antes de empacotar.')


def files(folder):
    entries = sorted(p for p in (portable / folder).iterdir() if p.is_file())
    if not entries:
        raise SystemExit(f'Pasta vazia: {folder}')
    return '\n'.join(
        f'      <File Id="f_{folder}_{hashlib.sha1(p.name.encode()).hexdigest()[:10]}" Name={quoteattr(p.name)} Source={quoteattr(str(p))}/>'
        for p in entries)


def component(ident, guid, body):
    # Per-machine components: the first file is the key path.
    return f'''    <Component Id="{ident}" Guid="{guid}" Win64="yes">
{body}
    </Component>'''


icon = quoteattr(str(root / 'packaging/windows/icon.ico'))
main = f'''      <File Id="f_exe" Name="DepoimentoStudio.exe" Source={quoteattr(str(exe))}/>
      <Shortcut Id="s_menu" Directory="ProgramMenuFolder" Name="Depoimento Studio" Target="[INSTALLDIR]DepoimentoStudio.exe" WorkingDirectory="INSTALLDIR" Icon="AppIcon"/>
      <Shortcut Id="s_desktop" Directory="DesktopFolder" Name="Depoimento Studio" Target="[INSTALLDIR]DepoimentoStudio.exe" WorkingDirectory="INSTALLDIR" Icon="AppIcon"/>'''
wxs = f'''<?xml version="1.0" encoding="utf-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
 <Product Id="*" Name="Depoimento Studio" Language="1033" Codepage="1252" Version="{version}" Manufacturer="Depoimento Studio" UpgradeCode="7C0E3B52-4E0B-4B37-9A55-2D6C1F0A9E41">
  <Package InstallerVersion="500" Compressed="yes" InstallScope="perMachine" Description="Depoimento Studio {version}"/>
  <MajorUpgrade DowngradeErrorMessage="Uma versão mais nova do Depoimento Studio já está instalada."/>
  <Media Id="1" Cabinet="app.cab" EmbedCab="yes"/>
  <Icon Id="AppIcon" SourceFile={icon}/>
  <Property Id="ARPPRODUCTICON" Value="AppIcon"/>
  <Property Id="ARPNOMODIFY" Value="1"/>
  <Property Id="ARPNOREPAIR" Value="1"/>
  <Directory Id="TARGETDIR" Name="SourceDir">
   <Directory Id="ProgramMenuFolder"/>
   <Directory Id="DesktopFolder"/>
   <Directory Id="ProgramFiles64Folder">
     <Directory Id="INSTALLDIR" Name="Depoimento Studio">
{component('main', 'B1F4C1E2-6A0D-4E55-8C0B-3E5D0D9A7F11', main)}
      <Directory Id="ToolsDir" Name="tools">
{component('tools', 'D0A6E7B3-2C41-4F7A-9B3E-6A1C8E2F5D22', files('tools'))}
      </Directory>
      <Directory Id="LicensesDir" Name="licenses">
{component('licenses', 'E3C9A1D4-7B52-4A68-8D1F-9B2E4C6A1F33', files('licenses'))}
      </Directory>
     </Directory>
   </Directory>
  </Directory>
  <Feature Id="App" Title="Depoimento Studio" Level="1">
   <ComponentRef Id="main"/>
   <ComponentRef Id="tools"/>
   <ComponentRef Id="licenses"/>
  </Feature>
 </Product>
</Wix>
'''
source = root / 'build/installer.wxs'
source.write_text(wxs, encoding='utf-8')
name = f'DepoimentoStudio-{version}' + ('-teste-local' if args.teste_local else '') + '.msi'
output = root / 'build' / name
subprocess.run([os.environ.get('WIXL', 'wixl'), '--arch', 'x64', '-o', str(output), str(source)], check=True)
digest = hashlib.sha256(output.read_bytes()).hexdigest()
(root / 'build' / (name + '.sha256')).write_text(f'{digest}  {name}\n')
print(f'{output} ({output.stat().st_size / 1048576:.1f} MB)\nsha256 {digest}')
