@echo off
rem SPDX-License-Identifier: GPL-2.0-only
rem
rem open-frontier read-only collection kit: SCANNER PC (Toshiba, Windows 2000).
rem
rem What it does: runs built-in Windows 2000 listing commands, and read-only
rem queries on the local order database if there is one, and writes the
rem output to a new folder on the USB stick this file runs from.
rem What it never does: talk to the scanner, change settings, delete files,
rem write to the registry or the database, open spool files, or copy
rem programs/drivers (.exe .dll .sys).
rem
rem Also works on a scanner PC with Windows XP, and with or without a server.
rem Run it from the USB stick (double click), as the usual user (FRONTIER),
rem when no scan is running. IMPORTANT: running this while a scan is in
rem progress can cause a sharing violation that aborts ExecLSeqObj.exe and
rem wedges the scanner.

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
mkdir "%OUT%\config-files\etc"

echo.
echo open-frontier: coleta somente leitura / read-only collection
echo Saida / output: %OUT%
echo Nao feche esta janela / do not close this window.
echo.

rem --- Basic identity ---
echo [1/13] Sistema / system...
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

rem --- Network, shares and open connections (spool and database traffic) ---
echo [2/13] Rede / network...
ipconfig /all > "%OUT%\network.txt" 2>&1
net share >> "%OUT%\network.txt" 2>&1
net use >> "%OUT%\network.txt" 2>&1
net config workstation >> "%OUT%\network.txt" 2>&1
net config server >> "%OUT%\network.txt" 2>&1
nbtstat -n >> "%OUT%\network.txt" 2>&1
arp -a >> "%OUT%\network.txt" 2>&1
route print >> "%OUT%\network.txt" 2>&1
netstat -an > "%OUT%\connections.txt" 2>&1
copy "%SystemRoot%\system32\drivers\etc\hosts" "%OUT%\config-files\etc\" > nul 2>&1
copy "%SystemRoot%\system32\drivers\etc\lmhosts" "%OUT%\config-files\etc\" > nul 2>&1

rem --- Running services ---
echo [3/13] Servicos / services...
net start > "%OUT%\services-running.txt" 2>&1

rem --- Driver files: names, dates and sizes only (no copies) ---
echo [4/13] Drivers (lista / list)...
dir /a "%SystemRoot%\system32\drivers" > "%OUT%\drivers-list.txt" 2>&1
dir /a "%SystemRoot%\inf" > "%OUT%\inf-list.txt" 2>&1

rem --- Registry: hardware enumeration, services, device classes ---
echo [5/13] Registro / registry (hardware)...
regedit /e "%OUT%\registry\enum.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Enum"
regedit /e "%OUT%\registry\services.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services"
regedit /e "%OUT%\registry\class.reg" "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Class"

rem --- Registry: Fujifilm-specific keys (spool paths live in HKLM\SOFTWARE\FUJIFILM\FDIA) ---
echo [6/13] Registro / registry (Fujifilm)...
regedit /e "%OUT%\registry\fujifilm.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\FUJIFILM"
regedit /e "%OUT%\registry\fujiphoto.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\Fuji Photo Film"
regedit /e "%OUT%\registry\frontier.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\Frontier"

rem --- Registry: SQL Server client aliases, ODBC data sources, programs started with Windows ---
echo [7/13] Registro / registry (SQL, ODBC, inicializacao / startup)...
regedit /e "%OUT%\registry\mssqlserver.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\MSSQLServer"
regedit /e "%OUT%\registry\odbc.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\ODBC"
regedit /e "%OUT%\registry\run-machine.reg" "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Run"
regedit /e "%OUT%\registry\run-user.reg" "HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run"

rem --- Registry: all installed software settings (may contain license keys) ---
echo [8/13] Registro / registry (software, opcional)...
regedit /e "%OUT%\OPTIONAL-may-contain-licenses\software.reg" "HKEY_LOCAL_MACHINE\SOFTWARE"

rem --- Spool folders: names, dates and sizes only. Listing never opens the files. ---
echo [9/13] InSpool, OutSpool, ImgExchange (lista / listing only)...
for %%S in (InSpool OutSpool ImgExchange) do (
  if exist "C:\%%S" dir /s /a "C:\%%S" > "%OUT%\spool-C-%%S.txt" 2>&1
  if exist "D:\%%S" dir /s /a "D:\%%S" > "%OUT%\spool-D-%%S.txt" 2>&1
)

rem --- File listings of both disks (names, dates, sizes only) ---
echo [10/13] Lista de arquivos / file listing (pode demorar / may take a while)...
dir /s /a C:\ > "%OUT%\files-C.txt" 2>&1
if exist D:\ dir /s /a D:\ > "%OUT%\files-D.txt" 2>&1
if exist "D:\Fujifilm\Frontier\Log" dir /a "D:\Fujifilm\Frontier\Log" > "%OUT%\log-folder.txt" 2>&1
for /d %%F in ("C:\Fuji*" "D:\Fuji*" "C:\Frontier*" "D:\Frontier*" "C:\Program Files\Fuji*" "C:\InSpool" "D:\InSpool" "C:\OutSpool" "D:\OutSpool") do tree /a "%%F" >> "%OUT%\folders-tree.txt" 2>&1

rem --- Text configuration files and text logs from Fujifilm/Frontier folders only ---
echo [11/13] Configuracao e logs / config files and logs (.ini .cfg .log .txt)...
for /d %%F in ("C:\Fuji*" "D:\Fuji*" "C:\Frontier*" "D:\Frontier*" "C:\Program Files\Fuji*" "C:\Program Files\Frontier*") do (
  for %%E in (ini cfg log txt) do xcopy "%%F\*.%%E" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
)

rem --- Order database: read-only SELECT queries (WITH NOLOCK) on the local SQL Server, if any ---
echo [12/13] Banco de ordens / order database (somente leitura / read-only)...
call :database

rem --- Full hardware/driver report (msinfo32 ships with Windows 2000) ---
echo [13/13] Relatorio de hardware / hardware report (msinfo32, alguns minutos / a few minutes)...
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

rem --- Read-only database queries. The second login is the factory default the Fuji software uses. ---
:database
set DB=%OUT%\database.txt
set OSQL=
for %%P in (osql.exe) do set OSQL=%%~$PATH:P
if not defined OSQL if exist "%ProgramFiles%\Microsoft SQL Server\80\Tools\Binn\osql.exe" set OSQL=%ProgramFiles%\Microsoft SQL Server\80\Tools\Binn\osql.exe
if not defined OSQL (
  echo osql.exe not found: no SQL Server tools on this machine > "%DB%"
  goto :eof
)
set Q1=SET NOCOUNT ON SELECT @@VERSION SELECT name FROM master..sysdatabases
set Q2=SET NOCOUNT ON SELECT name FROM sysobjects WHERE xtype='U' ORDER BY name SELECT * FROM DESCRIPTOR WITH (NOLOCK) SELECT * FROM DeviceTable WITH (NOLOCK) SELECT * FROM DeviceInfTable WITH (NOLOCK) SELECT * FROM DeviceStatusTable WITH (NOLOCK) SELECT * FROM ServiceTable WITH (NOLOCK)
set Q3=SET NOCOUNT ON SELECT TOP 30 * FROM OrderTable WITH (NOLOCK) ORDER BY FDIAManageID DESC SELECT COUNT(*) FROM ImageTable WITH (NOLOCK) SELECT TOP 20 * FROM ImageTable WITH (NOLOCK) ORDER BY FDIAManageID DESC
echo osql: %OSQL% > "%DB%"
echo === Windows login >> "%DB%"
"%OSQL%" -E -S (local) -l 10 -w 3000 -Q "%Q1%" >> "%DB%" 2>&1
"%OSQL%" -E -S (local) -l 10 -w 3000 -d FDIA_DB -Q "%Q2%" >> "%DB%" 2>&1
"%OSQL%" -E -S (local) -l 10 -w 3000 -d FDIA_DB -Q "%Q3%" >> "%DB%" 2>&1
echo === Fuji login >> "%DB%"
"%OSQL%" -U fdiaclient -P radiogaga -S (local) -l 10 -w 3000 -Q "%Q1%" >> "%DB%" 2>&1
"%OSQL%" -U fdiaclient -P radiogaga -S (local) -l 10 -w 3000 -d FDIA_DB -Q "%Q2%" >> "%DB%" 2>&1
"%OSQL%" -U fdiaclient -P radiogaga -S (local) -l 10 -w 3000 -d FDIA_DB -Q "%Q3%" >> "%DB%" 2>&1
goto :eof

:wrongdrive
echo.
echo ERRO: rode este arquivo a partir do PENDRIVE, nao de %DRV%
echo ERROR: run this file from the USB stick, not from %DRV%
pause

:end
endlocal
