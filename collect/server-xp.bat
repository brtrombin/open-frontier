@echo off
rem SPDX-License-Identifier: GPL-2.0-only
rem
rem open-frontier read-only collection kit: FRONT-END SERVER (Windows XP, also works on 7/10).
rem
rem What it does: runs built-in Windows listing commands, and read-only
rem queries on the local order database if there is one, and writes the
rem output to a new folder on the USB stick this file runs from.
rem What it never does: talk to the scanner, change settings, delete files,
rem write to the registry or the database, open spool files, or copy
rem programs/drivers (.exe .dll .sys).
rem
rem Run it from the USB stick (double click) when no scan is running.
rem IMPORTANT: running this while a scan is in progress can cause a sharing
rem violation that aborts ExecLSeqObj.exe and wedges the scanner.

setlocal
set DRV=%~d0

rem Refuse to run from the system disk: output must go to the USB stick.
if /i "%DRV%"=="C:" goto wrongdrive

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

echo [1/12] Sistema / system...
(
  echo open-frontier collection kit, front-end server
  echo Read-only. Contains machine configuration. Share privately, never publish.
  date /t
  time /t
  ver
  echo COMPUTERNAME=%COMPUTERNAME%
  echo USERNAME=%USERNAME%
) > "%OUT%\README.txt"
systeminfo > "%OUT%\system.txt" 2>&1
set >> "%OUT%\system.txt" 2>&1

echo [2/12] Rede / network...
ipconfig /all > "%OUT%\network.txt" 2>&1
net share >> "%OUT%\network.txt" 2>&1
net use >> "%OUT%\network.txt" 2>&1
net config workstation >> "%OUT%\network.txt" 2>&1
net config server >> "%OUT%\network.txt" 2>&1
nbtstat -n >> "%OUT%\network.txt" 2>&1
arp -a >> "%OUT%\network.txt" 2>&1
route print >> "%OUT%\network.txt" 2>&1
netstat -ano > "%OUT%\connections.txt" 2>&1
copy "%SystemRoot%\system32\drivers\etc\hosts" "%OUT%\config-files\etc\" > nul 2>&1
copy "%SystemRoot%\system32\drivers\etc\lmhosts" "%OUT%\config-files\etc\" > nul 2>&1

echo [3/12] Servicos, programas e drivers / services, programs and drivers...
net start > "%OUT%\services-running.txt" 2>&1
sc query type= service state= all > "%OUT%\services-all.txt" 2>&1
tasklist /svc > "%OUT%\programs-running.txt" 2>&1
driverquery /v > "%OUT%\drivers.txt" 2>&1
dir /a "%SystemRoot%\system32\drivers" > "%OUT%\drivers-list.txt" 2>&1

echo [4/12] Registro / registry (hardware)...
reg export "HKLM\SYSTEM\CurrentControlSet\Enum" "%OUT%\registry\enum.reg" > nul 2>&1
reg export "HKLM\SYSTEM\CurrentControlSet\Services" "%OUT%\registry\services.reg" > nul 2>&1
reg export "HKLM\SYSTEM\CurrentControlSet\Control\Class" "%OUT%\registry\class.reg" > nul 2>&1

echo [5/12] Registro / registry (Fujifilm)...
for %%K in ("HKLM\SOFTWARE\FUJIFILM" "HKLM\SOFTWARE\Fuji Photo Film" "HKLM\SOFTWARE\FUJI PHOTO FILM CO.,LTD." "HKLM\SOFTWARE\Frontier") do (
  reg query %%K /s >> "%OUT%\registry\fujifilm-keys.txt" 2>&1
)

echo [6/12] Registro / registry (SQL, ODBC, inicializacao / startup)...
reg export "HKLM\SOFTWARE\Microsoft\MSSQLServer" "%OUT%\registry\mssqlserver.reg" > nul 2>&1
reg export "HKLM\SOFTWARE\ODBC" "%OUT%\registry\odbc.reg" > nul 2>&1
reg export "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" "%OUT%\registry\run-machine.reg" > nul 2>&1
reg export "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" "%OUT%\registry\run-user.reg" > nul 2>&1

echo [7/12] Registro / registry (software, opcional)...
reg export "HKLM\SOFTWARE" "%OUT%\OPTIONAL-may-contain-licenses\software.reg" > nul 2>&1

echo [8/12] InSpool, OutSpool, ImgExchange (lista / listing only)...
for %%S in (InSpool OutSpool ImgExchange) do (
  if exist "C:\%%S" dir /s /a "C:\%%S" > "%OUT%\spool-C-%%S.txt" 2>&1
  if exist "D:\%%S" dir /s /a "D:\%%S" > "%OUT%\spool-D-%%S.txt" 2>&1
  if exist "E:\%%S" dir /s /a "E:\%%S" > "%OUT%\spool-E-%%S.txt" 2>&1
)

echo [9/12] Lista de arquivos / file listing (pode demorar / may take a while)...
dir /s /a C:\ > "%OUT%\files-C.txt" 2>&1
for %%D in (D E F) do (
  if /i not "%%D:"=="%DRV%" if exist %%D:\ dir /s /a %%D:\ > "%OUT%\files-%%D.txt" 2>&1
)

echo [10/12] Configuracao e logs / config files and logs (.ini .cfg .log .txt)...
for /d %%F in ("C:\Fuji*" "D:\Fuji*" "C:\Frontier*" "D:\Frontier*" "C:\Program Files\Fuji*" "C:\Program Files\Frontier*") do (
  for %%E in (ini cfg log txt) do xcopy "%%F\*.%%E" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
)

echo [11/12] Banco de ordens / order database (somente leitura / read-only)...
call :database

echo [12/12] Relatorio de hardware / hardware report (msinfo32, alguns minutos / a few minutes)...
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
