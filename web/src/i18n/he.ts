/** All user-facing strings (Hebrew). Keep copy here so components stay logic-only. */
import type { Cam, Decoder } from '../types';

export const camName: Record<Cam, string> = { F: 'קדמית', R: 'אחורית', C: 'פנימית' };
export const camLetter: Record<Cam, string> = { F: 'ק', R: 'א', C: 'פ' };

const CATEGORY: Record<string, string> = {
  normal: 'נסיעות',
  event: 'אירועים',
  parking: 'חניה',
  lapse: 'מואץ',
  timelapse: 'מואץ',
  photo: 'תמונות',
};
export const categoryName = (name: string) => {
  const leaf = name.split('/').pop() || name;
  return CATEGORY[leaf.toLowerCase()] ?? leaf;
};

export const plural = (n: number, one: string, many: string) => (n === 1 ? one : `${n} ${many}`);
export const clipsN = (n: number) => plural(n, 'קטע אחד', 'קטעים');
export const tripsN = (n: number) => plural(n, 'נסיעה אחת', 'נסיעות');

export const decoderLabel: Record<Decoder, string> = {
  auto: 'אוטומטי — מקורי, ומעבר ל-ffmpeg בעת הצורך',
  native: 'פענוח מקורי של הדפדפן בלבד',
  compat: 'תמיד המרה ל-H.264 עם ffmpeg',
};
export const decoderShort: Record<Decoder, string> = { auto: 'אוטו', native: 'מקורי', compat: 'תאימות' };

export const t = {
  appName: 'Dashcam Studio',
  tagline: 'כלי לצפייה נוחה של צילומי מצלמת דשבורד',
  kmh: 'קמ״ש',
  km: 'ק״מ',

  // welcome
  welcomeTitle: 'חברו את כרטיס המצלמה.',
  welcomeBody: 'הקלטות המצלמה הקדמית, האחורית והפנימית מותאמות אוטומטית ומתנגנות יחד. הקבצים רק נקראים — אף פעם לא משתנים.',
  rootFolder: 'תיקיית שורש',
  rootPlaceholder: 'E:\\  ·  /Volumes/DASHCAM  ·  /media/me/CARD',
  browse: 'עיון…',
  open: 'פתיחה',
  scanning: 'סורק…',
  cardDetected: 'זוהה כרטיס מצלמת רכב',
  backToPlayer: 'חזרה לנגן',

  // browse
  browseTitle: 'בחירת תיקיית מצלמה',
  thisComputer: 'המחשב הזה',
  up: 'למעלה',
  useFolder: 'בחירת תיקייה זו',
  dashcamHere: '✓ נמצאו כאן הקלטות',

  // top bar
  openOther: 'פתיחת תיקייה אחרת',
  rescan: 'סריקה מחדש',
  rescanned: 'התיקייה נסרקה מחדש',
  toggleLibrary: 'רשימת הקטעים (B)',
  toggleInspector: 'מפה ופרטים (I)',
  shortcuts: 'קיצורי מקלדת (?)',
  favorites: 'מועדפים',
  recordingType: 'סוג הקלטה',

  // library
  noRecordings: 'אין הקלטות בתיקייה הזו.',
  unknownDate: 'תאריך לא ידוע',
  favEmptyTitle: 'עדיין אין מועדפים',
  favEmptyBody: 'לחצו על ☆ ליד קטע (או על * בזמן צפייה) כדי לשמור אותו כאן.',
  favLoopWarn: 'המצלמה מקליטה בלולאה ודורסת קבצים ישנים. ייצאו את המועדפים כדי לשמור אותם לתמיד.',
  exportAllFavs: 'ייצוא כל המועדפים',
  favMissing: (n: number) => `${n} מועדפים כבר לא נמצאים בכרטיס`,
  selectTrip: 'בחירת כל הנסיעה',
  selectDay: 'בחירת כל היום',
  selectClip: 'בחירה',
  star: 'הוספה למועדפים (*)',
  unstar: 'הסרה מהמועדפים (*)',
  selected: (n: number) => (n === 1 ? 'נבחר קטע אחד' : `נבחרו ${n} קטעים`),
  exportSelected: 'ייצוא',
  starSelected: 'סימון כמועדפים',
  unstarSelected: 'הסרה מהמועדפים',
  clearSelection: 'ניקוי בחירה (Esc)',
  noCabin: 'ללא מצלמה פנימית',
  missingCam: (c: Cam) => `אין הקלטה ${camName[c]}`,

  // stage
  pickClip: 'בחרו קטע מהרשימה',
  makeMain: 'לתצוגה הראשית',
  notRecorded: (c: Cam) => `אין הקלטה ממצלמה ${camName[c]} לקטע הזה`,
  compatBadge: 'תאימות',
  compatTitle: 'הומר ל-H.264 עם ffmpeg כי הדפדפן לא מפענח את הקובץ המקורי',
  playFailed: (c: Cam, msg: string) => `לא ניתן לנגן את המצלמה ה${camName[c]} (${msg}).`,
  noHevc: 'הדפדפן לא מפענח HEVC (H.265). השתמשו ב-Chrome, Edge או Safari — או התקינו ffmpeg למצב תאימות.',
  compatSwitched: 'עבר לניגון במצב תאימות',
  compatSwitchedSub: 'הדפדפן לא מפענח את הווידאו — ffmpeg ממיר אותו בזמן אמת.',

  // transport
  trip: 'נסיעה',
  prevClip: 'הקטע הקודם (P)',
  nextClip: 'הקטע הבא (N)',
  back10: 'אחורה 10 שנ׳ (J)',
  fwd10: 'קדימה 10 שנ׳ (L)',
  playPause: 'ניגון / השהיה (רווח)',
  speed: 'מהירות ניגון ([ / ])',
  mute: 'השתקה (M)',
  volume: 'עוצמה',
  layoutFocus: 'מיקוד (G)',
  layoutGrid: 'רשת עם מפה (G)',
  layoutSolo: 'מצלמה אחת (G)',
  snapshot: 'צילום מסך של כל המצלמות (S)',
  download: 'הורדה',
  fullscreen: 'מסך מלא (F)',
  seek: 'מיקום בקטע',
  gpsSpeed: 'מהירות GPS',

  // download menu
  allCams: (n: number) => (n === 3 ? 'כל 3 המצלמות' : `${n} מצלמות`),
  camOnly: (c: Cam) => `${camName[c]} בלבד`,
  wholeTrip: 'כל הנסיעה',
  includeGpx: 'לכלול מסלול GPS (‎.gpx) בקובצי ZIP',
  downloading: 'ההורדה החלה',
  downloadingCams: (n: number) => (n === 1 ? 'מוריד מצלמה אחת כ-ZIP' : `מוריד ${n} מצלמות כ-ZIP`),
  downloadingCam: (c: Cam) => `מוריד את המצלמה ה${camName[c]}`,

  // export modal
  exportTitle: (n: number) => (n === 1 ? 'ייצוא קטע אחד' : `ייצוא ${n} קטעים`),
  exportCams: 'מצלמות',
  exportGpx: 'מסלול GPS (‎.gpx) לכל קטע',
  exportLayout: 'כל קטע נשמר בתיקייה משלו בתוך קובץ ה-ZIP.',
  exportTotal: 'גודל משוער',
  exportBig: 'קובץ גדול מ-4GB נשמר בפורמט ZIP64 — נתמך ב-Windows 10 ומעלה, macOS ולינוקס.',
  exportGo: 'הורדת ZIP',
  exportPreparing: 'מכין…',
  exportStarted: (n: number) => (n === 1 ? 'ייצוא קטע אחד החל' : `ייצוא ${n} קטעים החל`),
  exportNoCams: 'בחרו לפחות מצלמה אחת',
  exportSpan: 'טווח',
  cancel: 'ביטול',

  // inspector
  speedLbl: 'מהירות',
  heading: 'כיוון',
  clipDist: 'מרחק בקטע',
  topSpeed: 'מהירות שיא',
  noGps: 'אין GPS לקטע הזה',
  mapOffline: 'המפה לא זמינה — הטלמטריה עדיין פעילה',
  noFix: 'אין קליטת GPS',
  copyCoords: 'העתקת קואורדינטות',
  copied: 'הקואורדינטות הועתקו',
  openMaps: 'פתיחה ב-Google Maps',
  maps: 'מפות ↗',
  files: 'קבצים',
  decoder: 'מפענח',
  decoderSet: (d: Decoder) => `מפענח: ${decoderShort[d]}`,
  downloadCam: (c: Cam) => `הורדת המצלמה ה${camName[c]}`,

  // snapshot
  snapSaved: 'צילום המסך נשמר',
  snapFailed: 'צילום המסך נכשל',
  snapCams: (n: number) => (n === 1 ? 'מצלמה אחת' : `${n} מצלמות`),

  // help
  keyboard: 'מקלדת',
  keys: [
    [['רווח', 'K'], 'ניגון / השהיה'],
    [['←', '→'], 'דילוג 5 שנ׳ (Shift: שנייה)'],
    [['J', 'L'], 'דילוג 10 שנ׳'],
    [[',', '.'], 'פריים בודד'],
    [['P', 'N'], 'קטע קודם / הבא'],
    [['1', '2', '3'], 'קדמית / אחורית / פנימית לתצוגה הראשית'],
    [['G'], 'החלפת פריסה'],
    [['[', ']'], 'האטה / האצה'],
    [['M', '↑', '↓'], 'השתקה · עוצמה'],
    [['*'], 'סימון הקטע כמועדף'],
    [['S'], 'צילום מסך של כל המצלמות'],
    [['D'], 'הורדת כל המצלמות כ-ZIP'],
    [['F'], 'מסך מלא'],
    [['B', 'I'], 'הצגה / הסתרה של הרשימה / המפה'],
    [['E'], 'ניתוק המדיה / הוצאה בטוחה'],
    [['Home', 'End'], 'תחילת / סוף הקטע'],
    [['Esc'], 'סגירה · ניקוי בחירה'],
  ] as [string[], string][],
  keysNote: 'הקיצורים עובדים גם כשהמקלדת בעברית.',

  // media connection
  disconnect: 'ניתוק המדיה (E)',
  disconnectTitle: 'ניתוק המדיה',
  disconnectBody: 'הנגינה תיעצר והאפליקציה תשחרר את כל הקבצים בכרטיס, כך שאפשר לשלוף אותו בלי לפגוע בהקלטות.',
  disconnectEject: 'הוצאה בטוחה של הכונן',
  disconnectEjectHint: 'כמו "הוצאה" במערכת ההפעלה — מומלץ לפני שליפת הכרטיס.',
  disconnectNotRemovable: 'התיקייה לא נמצאת על כונן נשלף, לכן רק ננתק אותה מהאפליקציה.',
  disconnectGo: 'ניתוק',
  disconnectEjectGo: 'ניתוק והוצאה',
  disconnecting: 'משחרר את הקבצים…',
  disconnectedTitle: 'המדיה נותקה',
  safeToRemove: 'אפשר לשלוף את הכרטיס בבטחה',
  ejectFailed: 'המדיה נותקה, אבל מערכת ההפעלה לא הוציאה את הכונן',
  lostTitle: 'הכרטיס נותק',
  lostBody: 'חברו את הכרטיס מחדש — הנגינה תמשיך מאותה נקודה.',
  waiting: 'ממתין לכרטיס…',
  openOtherFolder: 'פתיחת תיקייה אחרת',
  reconnected: 'הכרטיס חובר מחדש',
  waitingForCard: 'חברו את כרטיס המצלמה — הוא יופיע כאן אוטומטית.',
  newCard: 'חדש',

  // errors
  openFailed: (m: string) => `לא ניתן לפתוח את הקטע: ${m}`,
  favSaveFailed: 'שמירת המועדפים נכשלה',
  starred: '★ נוסף למועדפים',
  unstarred: 'הוסר מהמועדפים',
};
