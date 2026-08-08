@echo off
REM ============================================================
REM  Likho - double click karo, blog chalu.
REM  Kuch install karne ki zarurat nahi hai. Sach me.
REM ============================================================

REM UTF-8 on karo, warna Hindi text aur emoji dabbe dabbe dikhte hain
chcp 65001 >nul 2>&1

title Likho - apna blog

REM Zaroori: batch file jahan hai wahin se chalao. Agar koi isko
REM shortcut se ya kisi aur folder se chalaye, toh bina iske exe nahi milega.
cd /d "%~dp0"

cls
echo.
echo   ===========================================
echo      Likho - apna blog, apne computer pe
echo   ===========================================
echo.

REM ---------- exe hai bhi ya nahi? ----------
if not exist "likho.exe" (
    echo   [X] likho.exe nahi mila.
    echo.
    echo   Aksar ye tab hota hai jab ZIP file ko "extract" nahi kiya gaya ho.
    echo   Windows zip ke andar se bhi file chalane deta hai, par tab ye
    echo   kaam nahi karega.
    echo.
    echo   Karo ye:
    echo     1. ZIP file pe right-click
    echo     2. "Extract All..." choose karo
    echo     3. Jo naya folder bane, usme se run.bat chalao
    echo.
    pause
    exit /b 1
)

REM ---------- port ----------
REM Default 4000. Agar wo busy ho toh aise chalao:  run.bat 4001
set PORT=4000
if not "%~1"=="" set PORT=%~1

echo   Blog chalu kiya ja raha hai...
echo   Browser apne aap khul jayega. 2-3 second lag sakte hain.
echo.
echo   ------------------------------------------------------
echo    PEHLI BAAR chalane pe Windows Firewall ka popup aa
echo    sakta hai. "Allow access" pe click karna.
echo.
echo    Wo sirf isliye hai taki tumhara phone bhi same wifi pe
echo    ye blog khol sake. Mana kar doge toh blog phir bhi
echo    chalega, bas phone wala feature band rahega.
echo   ------------------------------------------------------
echo.

REM ---------- chalao ----------
likho.exe -port %PORT%

REM ---------- yahan tab aate hain jab server band ho jaye ----------
REM %ERRORLEVEL% 0 nahi hai matlab kuch gadbad hui thi
if errorlevel 1 (
    echo.
    echo   ------------------------------------------------------
    echo    Kuch gadbad ho gayi. Upar likha error padho.
    echo.
    echo    Sabse common dikkat: port %PORT% pehle se busy hai.
    echo    Iska matlab ya toh Likho pehle se chal raha hai, ya
    echo    koi aur app us port pe hai.
    echo.
    echo    Fix: is window me ye type karo aur Enter dabao:
    echo         run.bat 4001
    echo   ------------------------------------------------------
    echo.
    pause
    exit /b 1
)

echo.
echo   Blog band ho gaya. Window band kar sakte ho.
echo   Tumhare saare posts "blog-data" folder me safe hain.
echo.
pause
