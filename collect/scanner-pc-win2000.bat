@echo off
rem SPDX-License-Identifier: GPL-2.0-only
rem
rem open-frontier read-only collection kit: SCANNER PC (Toshiba, Windows 2000).
rem
rem What it does: runs built-in Windows 2000 listing commands and writes their
rem output to a new folder on the USB stick this file runs from.
rem What it never does: talk to the scanner, change settings, delete files,
rem write to the registry, or copy programs/drivers (.exe .dll .sys).
rem
rem Run it from the USB stick (double click), as the usual user (FRONTIER),
rem when no scan is running.

setlocal
set DRV=%~d0

rem Refuse to run from the system or data disks: output must go to the USB stick.
if /i "%DRV%"=="C:" goto wrongdrive
if /i "%DRV%"=="D:" goto wrongdrive

set N=1
:findname
set OUT=%DRV%\collect-%COMPUTERNAME%-%N%
if exist "%OUT%\README.txt" (
  set /a N=N+1
  goto findname
)
mkdir "%OUT%"
mkdir "%OUT%\registry"
mkdir "%OUT%\OPTIONAL-may-contain-licenses"
mkdir "%OUT%\config-files"

echo.
echo open-frontier: coleta somente leitura / read-only collection
echo Saida / output: %OUT%
echo Nao feche esta janela / do not close this window.
echo.

rem --- Basic identity ---
echo [1/9] Sistema / system...
(
  echo open-frontier collection kit, scanner PC
  echo Read-only. Contains machine configuration. Share privately, never publish.
  date /t
  time /t
  ver
  echo COMPUTERNAME=%COMPUTERNAME%
  echo USERNAME=%USERNAME%
) > "%OUT%\README.txt"
ver > "%OUT%\system.txt"
set >> "%OUT%\system.txt" 2>&1

rem --- Network and shares (InSpool/OutSpool) ---
echo [2/9] Rede / network...
ipconfig /all > "%OUT%\network.txt" 2>&1
net share >> "%OUT%\network.txt" 2>&1
net use >> "%OUT%\network.txt" 2>&1
net config workstation >> "%OUT%\network.txt" 2>&1
net config server >> "%OUT%\network.txt" 2>&1

rem --- Running services ---
echo [3/9] Servicos / services...
net start > "%OUT%\services-running.txt" 2>&1

rem --- Driver files: names, dates and sizes only (no copies) ---
echo [4/9] Drivers (lista / list)...
dir /a "%SystemRoot%\system32\drivers" > "%OUT%\drivers-list.txt" 2>&1
dir /a "%SystemRoot%\inf" > "%OUT%\inf-list.txt" 2>&1

rem --- Registry: hardware enumeration, services, device classes ---
echo [5/9] Registro / registry (hardware)...
regedit /e "%OUT%\registry\enum.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Enum"
regedit /e "%OUT%\registry\services.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services"
regedit /e "%OUT%\registry\class.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Class"

rem --- Registry: all installed software settings (may contain license keys) ---
echo [6/9] Registro / registry (software, opcional)...
regedit /e "%OUT%\OPTIONAL-may-contain-licenses\software.reg" "HKEY_LOCAL_MACHINE\SOFTWARE"

rem --- File listings of both disks (names, dates, sizes only) ---
echo [7/9] Lista de arquivos / file listing (pode demorar / may take a while)...
dir /s /a C:\ > "%OUT%\files-C.txt" 2>&1
if exist D:\ dir /s /a D:\ > "%OUT%\files-D.txt" 2>&1
if exist "D:\Fujifilm\Frontier\Log" dir /a "D:\Fujifilm\Frontier\Log" > "%OUT%\log-folder.txt" 2>&1

rem --- Text configuration files from Fujifilm/Frontier folders only ---
echo [8/9] Arquivos de configuracao / config files (.ini .cfg)...
for /d %%F in ("C:\Fuji*" "D:\Fuji*" "C:\Frontier*" "D:\Frontier*" "C:\Program Files\Fuji*" "C:\Program Files\Frontier*") do (
  xcopy "%%F\*.ini" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
  xcopy "%%F\*.cfg" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
)

rem --- Full hardware/driver report (msinfo32 ships with Windows 2000) ---
echo [9/9] Relatorio de hardware / hardware report (msinfo32, alguns minutos / a few minutes)...
if exist "%CommonProgramFiles%\Microsoft Shared\MSInfo\msinfo32.exe" (
  start /wait "" "%CommonProgramFiles%\Microsoft Shared\MSInfo\msinfo32.exe" /report "%OUT%\msinfo32.txt"
) else (
  echo msinfo32.exe not found > "%OUT%\msinfo32.txt"
)

echo. >> "%OUT%\README.txt"
echo Finished: >> "%OUT%\README.txt"
date /t >> "%OUT%\README.txt"
time /t >> "%OUT%\README.txt"

echo.
echo PRONTO / DONE: %OUT%
echo Pode fechar a janela / you can close this window.
pause
goto end

:wrongdrive
echo.
echo ERRO: rode este arquivo a partir do PENDRIVE, nao de %DRV%
echo ERROR: run this file from the USB stick, not from %DRV%
pause

:end
endlocal
