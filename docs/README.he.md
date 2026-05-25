# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · **עברית**

<div dir="rtl">

כלי קל לנגינה אוטומטית של מקשים (rapid-fire / turbo) ל-Windows עם ממשק גרפי מקורי. קשרו מקש כך שיחזור על עצמו — או על מקש אחר — אוטומטית, בין אם בזמן החזקה ובין אם כמתג הפעלה/כיבוי. לכל כלל יש מרווח משלו.

עובד עם יישומים רגילים ועם משחקים: המקשים מוזרקים כקודי סריקה (**scan codes**) של חומרה דרך `SendInput` (כך שמשחקי DirectInput מקבלים אותם), וכל לחיצה מוחזקת לרגע קצר כדי שמשחקים מבוססי-דגימה ידגמו אותה באמינות.

> Windows בלבד. נבנה עם Go + [lxn/walk](https://github.com/lxn/walk) (פקדי Win32 מקוריים), קובץ הרצה יחיד ועצמאי, ללא סביבת ריצה להתקנה.

</div>

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

<div dir="rtl">

## הורדה

השיגו את `turbokey.exe` העדכני מעמוד [Releases](../../releases). כל דחיפה (push) ל-`main` בונה ומפרסמת אוטומטית גרסה חדשה ממוספרת דרך GitHub Actions.

## תכונות

- לכל כלל: **מקש הפעלה**, **מקש פלט** (ברירת מחדל: מקש ההפעלה), **מצב**, **מרווח** והפעלה/השבתה.
- מקשי מקלדת **או לחצני עכבר** (שמאל / ימין / אמצעי / X1 / X2) כהפעלה וכפלט.
- שני מצבים לכל כלל:
  - **החזקה** — חוזר כל עוד מקש ההפעלה לחוץ פיזית.
  - **החלפה** — לחיצה אחת מתחילה את החזרה, הבאה עוצרת אותה.
- מתג ראשי גלובלי עם מקש קיצור **F8**.
- אפשר להגביל את הנגינה המהירה ליישומים מסוימים (לפי שם תהליך); ריק = עובד בכל מקום.
- **מגש המערכת**: סגירת החלון ממזערת אותו למגש (הכלי ממשיך לרוץ). לחיצה שמאלית על הסמל משחזרת אותו; לחיצה ימנית לתפריט שמציג את החלון, מחליף את המתג הראשי, או יוצא.
- **הפעלה עם Windows** אופציונלית, רשומה כמשימה מתוזמנת כדי להתחיל בהרשאות מוגברות בכניסה ללא בקשת UAC.
- הכללים נשמרים ב-`config.json` ליד קובץ ההרצה ונטענים מחדש בהפעלה.
- ממשק מתורגם ל-13 שפות (כולל ערבית ועברית מימין לשמאל בפריסה ממוראה לחלוטין), מזוהה אוטומטית ממערכת ההפעלה, עם בורר בתוך היישום.
- הכלי מסנן את הקלט הסינתטי שלו עצמו, כך שלעולם אינו מפעיל את עצמו מחדש.

## איך זה עובד

- וו מקלדת ברמה נמוכה `WH_KEYBOARD_LL` מזהה את הלחיצות שלכם ובולע את מקש ההפעלה המוגדר, כך שהמשחק רואה רק את הפולסים החוזרים הנקיים.
- כל פולס הוא `מקש למטה → החזקה ~30 מ"ש → מקש למעלה`, נשלח עם `KEYEVENTF_SCANCODE`. ההחזקה הכרחית: משחק מבוסס-דגימה דוגם את מצב המקשים בכל פריים והיה מפספס זוג למטה/למעלה שנופל בין שתי דגימות.
- אירועים סינתטיים מתויגים דרך `dwExtraInfo`, כך שהוו מזהה את המקשים שהוא עצמו הזריק ומעביר אותם במקום לפעול עליהם.

## דרישות

- Windows 10/11 (x64).
- **הרשאות מנהל.** קובץ ההרצה מבקש העלאת הרשאות אוטומטית (בקשת UAC בהפעלה). זה נדרש כי UIPI של Windows חוסם קלט מוזרק מתהליך לא-מורם מלהגיע לחלונות מורמים — והרבה משחקים רצים בהרשאות מוגברות.

## בנייה

לא נדרש CGO, כך שאפשר לקמפל-צולב ל-Windows מ-Linux/WSL או לבנות באופן מקורי ב-Windows.

</div>

```sh
# Linux / WSL (קימפול צולב) או Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

<div dir="rtl">

הסקריפט מתקין את `rsrc` כדי להטמיע את מניפסט היישום (Common Controls v6, מודעות DPI, requireAdministrator), ואז מריץ `go build`. לבנייה ידנית:

</div>

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

<div dir="rtl">

## מבנה הפרויקט

</div>

```
cmd/turbokey/      נקודת כניסה (main) + מניפסט יישום Windows + סמל
cmd/gen-icon/      מחולל סמלים בזמן בנייה (מרנדר icon.ico / icon.png)
internal/keys/     טבלת מקשים ולחצני עכבר; מיפוי שם <-> קוד מקש וירטואלי
internal/i18n/     תרגום הממשק; קטלוגי JSON מוטמעים (locales/)
internal/config/   מודל הכללים וטעינה/שמירה של config.json
internal/winput/   עוטפי Win32: ווי מקלדת ועכבר, SendInput, חזית
internal/engine/   מנוע הנגינה המהירה: ניתוב ווים, מתג ראשי, עובדים
internal/ui/       ממשק Win32 מקורי (lxn/walk)
internal/autostart/ משימה מתוזמנת בכניסה (הפעלה עם Windows)
internal/buildinfo/ גרסת בנייה, מוזרקת דרך -ldflags
```

<div dir="rtl">

## שימוש

1. הפעילו את `turbokey.exe` ואשרו את בקשת ה-UAC.
2. בעורך, בחרו **מקש הפעלה**, **מקש פלט** (ברירת מחדל = כמו ההפעלה), **מצב**, ו**מרווח (מ"ש)**, ואז לחצו **הוסף**.
3. לשינוי כלל, לחצו עליו (ערכיו נטענים לעורך), ערכו ולחצו **עדכון**. לחיצה כפולה על שורה מפעילה/משביתה אותה; **מחק** מסיר אותה.
4. לחצו **F8** (או סמנו את המתג הראשי) להפעלה, ואז לחצו על מקש ההפעלה כדי לירות. לחצו **F8** שוב כשסיימתם.

הערות:
- המרווח הוא הפער *בין* הלחיצות; כל לחיצה גם מחזיקה את המקש ~30 מ"ש, כך שהתקרה המעשית היא כ-25-30 לחיצות לשנייה. זה בשפע לכל מיומנות במשחק — ללחוץ מהר יותר לא עוזר, כי המשחק דוגם בכל פריים.
- כל עוד המתג הראשי דלוק, מקשי ההפעלה המוגדרים נתפסים ב**כל** היישומים, לא רק במשחקים. כבו אותו (F8) כשאינכם משתמשים.
- F8 שמור כמקש קיצור ראשי כל עוד הכלי רץ.

## תצורה

`config.json` (נוצר ליד קובץ ההרצה) קריא לבני אדם:

</div>

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

<div dir="rtl">

- `output: ""` פירושו "כמו מקש ההפעלה".
- `mode`: `"hold"` או `"toggle"`.
- שמות מקשים: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`, ולחצני עכבר `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (אלה מזהים נייטרליים-שפה ואינם מתורגמים, כך שהתצורה נשארת תקפה בין שפות הממשק.)

## הגבלה ליישומים מסוימים

כברירת מחדל הנגינה המהירה פעילה בכל יישום. כדי להגביל אותה, מלאו את שדה "יישומים פעילים" בשם תהליך אחד או יותר (למשל `DNFGame.exe`, מופרדים בפסיקים). הנגינה המהירה תופעל אז רק כשאחד מהיישומים האלה בחזית; במקומות אחרים המקשים שלכם מתנהגים כרגיל.

לא יודעים את שם התהליך? לחצו **בחר יישום…** ובחרו אותו מרשימת התוכניות הרצות (מוצגות לפי כותרת חלון וקובץ הרצה). מקש הקיצור הראשי F8 תמיד עובד, ללא תלות ביישום הפעיל.

## שפה

בחרו את השפה מהתפריט הנפתח בפינה הימנית-עליונה של החלון. הבחירה נשמרת ב-`config.json` והיישום מופעל מחדש כדי להחיל אותה; "אוטומטי" עוקב אחר שפת הממשק של Windows.

שפות מצורפות: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

שפות מימין לשמאל (ערבית, עברית) ממראות את כל פריסת החלון דרך `WS_EX_LAYOUTRTL` (המאפיין `RightToLeftLayout` של walk).

סדר ההכרעה: משתנה הסביבה `TURBOKEY_LANG` (`zh`/`en`), אחר כך הבחירה השמורה, אחר כך שפת מערכת ההפעלה.

קטלוגי ההודעות הם JSON רגיל תחת `internal/i18n/locales/`, מוטמעים עם `go:embed`. כדי להוסיף שפה, הניחו `internal/i18n/locales/<code>.json` (העתיקו את `en.json` ותרגמו את הערכים) ובנו מחדש.

## אזהרות

- וו מקלדת גלובלי בתוספת הזרקת קלט עלולים לעורר התראות שווא של אנטי-וירוס.
- חלק מהמשחקים המקוונים אוסרים מאקרו/אוטומציה בתנאי השירות שלהם, ומערכות אנטי-צ'יט עלולות לזהות או לחסום קלט סינתטי. השתמשו באחריות ועל אחריותכם בלבד.

## רישיון

[MIT](../LICENSE)

</div>
