# sources/henry-sermons: what was read and what was not

<!-- toc: none -->

Made by hand on 2026-11-05 from the Internet Archive's OCR (`<id>_djvu.txt` and `<id>_djvu.xml`, kept in the git-ignored `.archive-cache/`; nothing of it is committed). No tool wrote `rows.jsonl`: the contents tables and the printed heads of the sermons were read, and a row is a title, the preaching text as the printed head gives it, the Archive's reader page and, where printed, a date. No sentence of a sermon is in this repository.

## Counts

51 rows; 45 with a preaching text (`ref`), 6 without; 27 with a date; 3 carry `notes` containing `unsure:`. (The first version had 52: `funeral-sermon-4` is out, see "Rows left out because the scan cannot place them".)

| Edition | Item | Rows |
| --- | --- | --- |
| *The Miscellaneous Works of the Rev. Matthew Henry*, J. O. Robinson, London, 1830 | `miscellaneouswor00henr` | 44 |
| *The Complete Works of the Rev. Matthew Henry*, vol. 1, A. Fullarton, Edinburgh, 1847 | `completeworksofr0001revm` | 7 |

The 1847 rows are the discourses of the Complete Works that print their text in a clear head: Sober-Mindedness, A Church in the House, Friendly Visits, Great Britain's Joys (the former sermon), the Work and Success of the Ministry, and the funeral sermons for Benion and Tallents. Everything else was read in the 1830 book, which holds all of those too and the manuscript sermons first printed in it. The 1833 second printing (`miscellaneouswo02henrgoog`) has the same contents as the 1830 book; it was looked at, not used.

By kind: 6 sermons of The Worth of the Soul (one row each, all Matthew 16:26: the six are six rows because each is a separate sermon with its own page and doctrine, and a reader opening Matthew 16:26 should get six links, not one), 3 funeral and 4 fast sermons from the manuscripts, 1 sermon of "Sermons and Charges", 7 funeral sermons printed by Henry or from his papers (Owen, Benion, Tallents, Lawrence, Stretton, Burgess, and the sermon at Broad Oak for his mother), 3 sermons of the Directions for Daily Communion with God, the four discourses against vice and profaneness, and 23 single sermons and discourses (51 rows). Matthew 16:26 answers `worth-of-the-soul-1` to `-6` (the test asserts it).

## How a row's page was found

`url` is `https://archive.org/details/<item>/page/n<N>`, and N is the reader's leaf, counted from 0. The first version of this file got N wrong for the 1830 book: on 2026-11-06 the Governor's phone, opening "The Worth of the Soul, Sermon I" (n1254), landed on printed page 1217, headed "SERMON II.", in the middle of the second sermon, and 44 of the 45 urls of the 1830 book were then found to be 1 to 6 leaves late (14 by 2, 14 by 4, 15 by 6, 1 by 1; the 45th, `funeral-sermon-4`, has no leaf in the scan). The 7 urls of the 1847 volume were right. The Archive's reader does not show every scanned leaf (covers and colour cards are left out), so neither a page name in `_djvu.xml`, nor the leaf number in `_scandata.xml`, nor a printed page is the reader's N.

Method. For each item the Archive's `<item>_djvu.xml` (the OCR with one `OBJECT` a leaf the reader shows) and `<item>_scandata.xml` (which leaves are in the reader: `addToAccessFormats`) were downloaded. The reader's leaf N is the 0-based index of the `OBJECT` in `_djvu.xml`: 1,462 objects for the 1830 book (1,470 scanned leaves less 8 not in the reader) and 654 for the 1847 volume (656 less 2). That was proved on the report that started this fix: leaf 1254 of `miscellaneouswor00henr` reads "SERMON II." over printed page 1217, the page the Governor's phone opened for "The Worth of the Soul, Sermon I". Then, for each row, the leaf whose text prints the sermon's head (the title lines and the preaching text, or "SERMON N." and its text line) was found by reading the OCR of the leaves round the old guess and of every leaf where the title's words stood; where the head stands in the middle of a leaf (the second Self-Consideration sermon, England's Hopes, the funeral and fast sermons of the manuscripts) that leaf is the one. The printed page is the number the OCR reads on the leaf and its neighbours, voted over the 8 leaves each side (the OCR garbles single numbers: "1316" for 1216), and it agrees with the printed pages `_scandata.xml` asserts where it asserts one. It is for reading only; nothing uses it.

The check. `TestHenryLeavesHoldTheirHeads` (internal/sources/load_test.go) reads, for each row, the OCR of the leaf its url names and requires every phrase of the last column of the table below, compared in lower case with everything but letters and digits made a single space (so line breaks and the OCR's stray punctuation do not matter, and "sermon i" does not match "SERMON II."). It also requires that the leaf before does not print all the phrases: a head that stands on two leaves in a row is a running head, and the sermon began earlier. A phrase is one the OCR really reads on that leaf; where the OCR garbles a word of the head the phrase is another part of the head. The test needs the OCR, which is not in the repository: `sh scripts/fetch-henry-ocr.sh` downloads `miscellaneouswor00henr_djvu.xml` (115 MB) and `completeworksofr0001revm_djvu.xml` (39 MB) into the git-ignored `.archive-cache/` (or `SERMON_INDEX_ARCHIVE_CACHE`); when they are there `go test ./...` runs the check, and when they are not it is skipped and says so. `make henry-leaves` fetches them and runs just that test. The test `TestHenryLeafTable` runs always: the table below and `rows.jsonl` must name the same rows and the same leaves.

| Row | Item | Leaf N | Printed page | Head (all of these phrases are on the leaf) |
| --- | --- | --- | --- | --- |
| `sober-mindedness` | `completeworksofr0001revm` | 73 | 58 | sober mindedness pressed upon young people; titus |
| `church-in-the-house` | `completeworksofr0001revm` | 263 | 248 | a church in the house; family religion |
| `friendly-visits` | `completeworksofr0001revm` | 283 | 268 | right management; of friendly visits |
| `britains-joys` | `completeworksofr0001revm` | 459 | 444 | great britain s present joys and hopes; in two sermons |
| `work-and-success-of-the-ministry` | `completeworksofr0001revm` | 486 | 471 | work and success of the ministry; tuesday lecture |
| `funeral-benion` | `completeworksofr0001revm` | 582 | 567 | funeral of dr samuel benion; all flesh is as grass |
| `funeral-tallents` | `completeworksofr0001revm` | 615 | 600 | funeral of the rev mr francis tallents |
| `katharine-henry-funeral` | `miscellaneouswor00henr` | 201 | 164 | death of mrs katharine henry; proverbs xxxi 28 |
| `pleasantness-of-a-religious-life` | `miscellaneouswor00henr` | 262 | 225 | pleasantness of a religious life |
| `meekness-and-quietness-of-spirit` | `miscellaneouswor00henr` | 305 | 268 | concerning meekness and quietness of spirit |
| `christianity-no-sect` | `miscellaneouswor00henr` | 351 | 314 | the christian religion is not a sect; acts xxviii 22 |
| `daily-communion-1` | `miscellaneouswor00henr` | 465 | 428 | daily communion with god; the first discourse; psalm v 3 |
| `daily-communion-2` | `miscellaneouswor00henr` | 476 | 439 | the second discourse; how to spend the day with god |
| `daily-communion-3` | `miscellaneouswor00henr` | 489 | 452 | the third discourse; psalm iv 8 |
| `friendly-admonition` | `miscellaneouswor00henr` | 504 | 467 | a friendly admonition; drunkards and tipplers |
| `word-of-advice` | `miscellaneouswor00henr` | 517 | 480 | a word of advice; wanton and unclean |
| `serious-address` | `miscellaneouswor00henr` | 529 | 492 | a serious address; profane the lord s day |
| `check-to-an-ungoverned-tongue` | `miscellaneouswor00henr` | 542 | 505 | a check to an ungoverned tongue |
| `folly-of-despising-our-own-souls` | `miscellaneouswor00henr` | 556 | 519 | the folly of despising our own souls; proverbs xv 32 |
| `folly-of-despising-our-own-ways` | `miscellaneouswor00henr` | 575 | 538 | the folly of despising our own ways; proverbs xix 16 |
| `reformation-of-manners` | `miscellaneouswor00henr` | 637 | 600 | societies for reformation of manners; psalm vii 9 |
| `popery-a-spiritual-tyranny` | `miscellaneouswor00henr` | 652 | 615 | popery a spiritual tyranny; isaiah li 23 |
| `englands-hopes` | `miscellaneouswor00henr` | 782 | 745 | england s hopes |
| `promises-of-god` | `miscellaneouswor00henr` | 806 | 769 | the promises of god; 2 cor vii 1 |
| `disputes-reviewed` | `miscellaneouswor00henr` | 811 | 774 | disputes reviewed; mark ix 33 |
| `faith-in-christ` | `miscellaneouswor00henr` | 821 | 784 | faith in christ inferred from faith in god; john xiv 1 |
| `forgiveness-of-sin` | `miscellaneouswor00henr` | 835 | 798 | as a debt; matt vi 12 |
| `hope-and-fear-balanced` | `miscellaneouswor00henr` | 850 | 813 | hope and fear balanced; psalm cxlvii 11 |
| `catechising-of-youth` | `miscellaneouswor00henr` | 863 | 826 | the catechising of youth |
| `memorial-of-the-fire-of-the-lord` | `miscellaneouswor00henr` | 876 | 839 | a memorial of the fire of the lord; numbers xi 3 |
| `christs-favour-to-little-children` | `miscellaneouswor00henr` | 966 | 929 | favour to little children; mark x 16 |
| `atkinson-ordination` | `miscellaneouswor00henr` | 1004 | 967 | ordination of mr atkinson; isaiah vi 8 |
| `funeral-owen` | `miscellaneouswor00henr` | 1028 | 991 | funeral of the rev james owen; acts xx 37 38 |
| `funeral-lawrence` | `miscellaneouswor00henr` | 1092 | 1055 | funeral of mr samuel lawrence; lest i should have sorrow upon sorrow |
| `funeral-stretton` | `miscellaneouswor00henr` | 1109 | 1072 | death of the rev richard stretton |
| `funeral-burgess` | `miscellaneouswor00henr` | 1127 | 1090 | death of the rev daniel burgess; 2 corinthians iv 7 |
| `new-meeting-house-chester` | `miscellaneouswor00henr` | 1169 | 1132 | new meeting house at chester; august 8 1700 |
| `worth-of-the-soul-1` | `miscellaneouswor00henr` | 1248 | 1211 | the worth of the soul; in six sermons; sermon i; what is a man |
| `worth-of-the-soul-2` | `miscellaneouswor00henr` | 1252 | 1215 | sermon ii; what is a man profited |
| `worth-of-the-soul-3` | `miscellaneouswor00henr` | 1256 | 1219 | sermon iii; what is a man profited |
| `worth-of-the-soul-4` | `miscellaneouswor00henr` | 1260 | 1223 | sermon iv; what is a man profited |
| `worth-of-the-soul-5` | `miscellaneouswor00henr` | 1264 | 1227 | sermon v; what is a man profited |
| `worth-of-the-soul-6` | `miscellaneouswor00henr` | 1268 | 1231 | sermon vi; what is a man profited |
| `sermons-and-charges` | `miscellaneouswor00henr` | 1273 | 1236 | sermons and charges; old testament institutions are new testament |
| `funeral-sermon-1` | `miscellaneouswor00henr` | 1285 | 1248 | funeral sermons; sermon i; isaiah xxxviii 12 |
| `funeral-sermon-2` | `miscellaneouswor00henr` | 1286 | 1249 | sermon ii; job xiv 10 |
| `funeral-sermon-3` | `miscellaneouswor00henr` | 1288 | 1251 | genesis xlix 18; february 2nd 1702 |
| `fast-sermon-1` | `miscellaneouswor00henr` | 1290 | 1255 | fast sermons; sermon i; isaiah viii 12 13 |
| `fast-sermon-2` | `miscellaneouswor00henr` | 1296 | 1259 | sermon ii; isaiah xlii 21 |
| `fast-sermon-3` | `miscellaneouswor00henr` | 1300 | 1263 | jeremiah xiv 7; june 10 1702 |
| `fast-sermon-4` | `miscellaneouswor00henr` | 1304 | 1267 | sermon iv; mark xiii 7 |

The leaves are the first of a sermon's head; a few notes on what the table does not show:

- `fast-sermon-1`: the scan holds the book's printed pages 1254 and 1255 twice, as leaves 1289 and 1291 and as leaves 1290 and 1292 (Fast Sermon I begins on printed page 1255); the row uses the first, 1290.
- `folly-of-despising-our-own-ways`, `englands-hopes`, `funeral-sermon-2`, `funeral-sermon-3` and the fast sermons begin in the middle of a leaf, after the end of the sermon before.
- `daily-communion-1` is the title leaf of the three discourses, which prints the first discourse's text (Psalm 5:3) too. `friendly-admonition` is n504, where its own title is printed, not n503, the title leaf of the four discourses against vice and profaneness (which holds the preface "To the Reader" of all four).
- `sermons-and-charges`, `pleasantness-of-a-religious-life` and `meekness-and-quietness-of-spirit`: the title leaf, with the preface "To the Reader" that comes before the discourse.

## Rows left out because the scan cannot place them

- `funeral-sermon-4` (Funeral Sermon IV, Psalm 90:6): the 1830 scan lacks the leaves that print its head. The leaf of Funeral Sermon III (n1288, printed page 1251) is followed by one printed 1254 (n1289, which reads like the end of a sermon at a young man's funeral: "Be not vain... I have found praying hours the sweetest hours"), so the printed pages 1252 and 1253 are not in the scan, and the OCR holds no "Psalm xc. 6" nor "Sermon IV" for the funeral sermons. The first version of this file gave it n1295, a leaf in the middle of Fast Sermon II. The row was taken out of the index rather than left wrong. The 1833 second printing (`miscellaneouswo02henrgoog`, a Google scan) prints it, headed "SERMON IV." with "Psalm xc. 6" and the funeral of Mr. Benjamin Cluby (13 January 1704); using it would add a third item to the source and has not been done.

## What the OCR could not place

The 1830 contents table (leaf 14 of the item) came out as two columns read together and is scrambled in three places, and the 1847 contents (leaf 8) lost its page numbers and several words:

- 1830: the four discourses against vice and profaneness (the lines `l¥T ^lol*V»ntll l-\ O V I M (IT TA/ 'Pl./\T1T1A` and `xii. o(imjiii.u~L)reaKiiig. i v • irruiciiie`), the funeral sermons for Lawrence and Stretton and the lives that follow them (page numbers `luoo`, `1 109`, `X Itllddl Ul Vf 11 \JH XIX 1 .UClWl^ll^K^`), and the page numbers of "Sober-mindedness" and "A church in the house" (`Ol o`, `OoO`). They were found by reading the body for their title lines, not from the table.
- 1847: "The Pleasantness of a Religious Life ... and improved; and recommended to the consideration of all, gh of Young People" and "Sober-Mindedness presse ae Wisse Per ple" are cut and joined with their neighbours; the page numbers of A Friendly Admonition, The Communicants' Companion, the Directions and Self-consideration are interleaved. Again they were located in the body.
- The 1830 contents entry "Separation without rebellion" (printed page 1132) is the sermon at the opening of the new meeting-house at Chester (Joshua xxii. 22, 23): a title the contents gives and the head does not. It is one row, `new-meeting-house-chester`.

The six rows without a text (`friendly-admonition`, `word-of-advice`, `serious-address`, `check-to-an-ungoverned-tongue`, `pleasantness-of-a-religious-life`, `meekness-and-quietness-of-spirit`) have no text in the printed head and say `no text printed`: they are discourses with a preface "To the Reader" and headings by section. The text of the funeral and fast sermons, of the Worth of the Soul and of the single sermons is printed under the title and was read from there; where the OCR garbled a reference it is said in `notes` (`catechising-of-youth`, the first Worth of the Soul head, `funeral-sermon-4`, `funeral-stretton`, `funeral-owen`, `fast-sermon-1`).

## Left out on purpose

- Philip Henry's 41 sermons of the 1830 Appendix ("What Christ is made to believers in forty real benefits") and his Life: the father's, not Matthew's.
- The funeral sermons on Matthew Henry by William Tong, John Reynolds and Daniel Williams: preached by others.
- The treatises that are not sermons: The Communicants' Companion, the Treatise on Baptism, A Method for Prayer, the Family Hymns and Essay on Psalmody, the Brief Inquiry into the Nature of Schism, the Layman's Reasons, the catechisms, the lives and memoirs (Benion, Tallents, Lawrence, Stretton, Burgess, Illidge, Tylston, Radford, Hulton), and the prefaces.
- The ordination charges (Charges I to IV and the one the OCR reads as "CHARGE VL" in "Sermons and Charges", and the Exhortation at Mr. Clark's ordination): charges with no text, printed after the Numbers xv. 39 sermon. They are the part of the probe's "about 20 numbered manuscript sermons" that is not a row: 6 + 3 + 4 + the Numbers sermon = 14 rows, and the charges are the rest.
- The 1782 Select Sermons and the single sermons of 1704 to 1735 on the Archive (`bim_eighteenth-century_...`), and the 1979 Baker reprint (a lending copy): not used.

## Sweep: three more preachers on the Archive as scans with printed texts

Candidates for the add-a-source recipe (docs/sources.md), not built. Each was checked for the OCR route (`<id>_djvu.txt` present, `possible-copyright-status` NOT_IN_COPYRIGHT where the Archive gives one) and for a printed text under each sermon's title; none was read beyond the first few sermons.

| Preacher | Item | What it is | Printed text | Notes |
| --- | --- | --- | --- | --- |
| George Whitefield (1714-1770) | `eighteensermonsp00whit` | *Eighteen sermons preached by the late Rev. George Whitefield*, 1771 | `SERMON I.` then the title, then the text (`Revel. xxii. 21`, `Psalm xlvi. 1-6`, `3 Ep. John ver. 2`) | the OCR keeps the long s (`Minifter`), so titles need care; the six-volume *Works* (`worksofreverendg01whit` to `06whit`, 1771) hold more |
| Jonathan Edwards (1703-1758) | `worksofpresident08edwa` | *The works of President Edwards*, Worcester, 1808, vol. 8: "Sermons on various important subjects", numbered (Sermon XVI onward in the OCR) | `SERMON XVI.`, the title, then the text in capitals (`HEBREWS v. 12.`) | status NOT_IN_COPYRIGHT; other volumes of the set (`worksofpresident02edwa` to `07edwa`) hold treatises and more sermons |
| John Wesley (1703-1791) | `sermonsonseveral01wesl` | *Sermons on several occasions*, New York, J. & J. Harper, 1831, vol. 1 (the contents give each sermon's title and text: "Salvation by Faith; Ephesians ii. 8") | the contents table prints title and text on one line | NOT_IN_COPYRIGHT; `standardsermonsc01wesluoft` (a 1951 reprint) shows no status and is not recommended |
