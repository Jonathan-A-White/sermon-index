# sources/henry-sermons: what was read and what was not

Made by hand on 2026-11-05 from the Internet Archive's OCR (`<id>_djvu.txt` and `<id>_djvu.xml`, kept in the git-ignored `.archive-cache/`; nothing of it is committed). No tool wrote `rows.jsonl`: the contents tables and the printed heads of the sermons were read, and a row is a title, the preaching text as the printed head gives it, the Archive's reader page and, where printed, a date. No sentence of a sermon is in this repository.

## Counts

52 rows; 46 with a preaching text (`ref`), 6 without; 28 with a date; 5 carry `notes` beginning `unsure:`.

| Edition | Item | Rows |
| --- | --- | --- |
| *The Miscellaneous Works of the Rev. Matthew Henry*, J. O. Robinson, London, 1830 | `miscellaneouswor00henr` | 45 |
| *The Complete Works of the Rev. Matthew Henry*, vol. 1, A. Fullarton, Edinburgh, 1847 | `completeworksofr0001revm` | 7 |

The 1847 rows are the discourses of the Complete Works that print their text in a clear head: Sober-Mindedness, A Church in the House, Friendly Visits, Great Britain's Joys (the former sermon), the Work and Success of the Ministry, and the funeral sermons for Benion and Tallents. Everything else was read in the 1830 book, which holds all of those too and the manuscript sermons first printed in it. The 1833 second printing (`miscellaneouswo02henrgoog`) has the same contents as the 1830 book; it was looked at, not used.

By kind: 6 sermons of The Worth of the Soul (one row each, all Matthew 16:26: the six are six rows because each is a separate sermon with its own page and doctrine, and a reader opening Matthew 16:26 should get six links, not one), 4 funeral and 4 fast sermons from the manuscripts, 1 sermon of "Sermons and Charges", 7 funeral sermons printed by Henry or from his papers (Owen, Benion, Tallents, Lawrence, Stretton, Burgess, and the sermon at Broad Oak for his mother), 3 sermons of the Directions for Daily Communion with God, the four discourses against vice and profaneness, and 23 single sermons and discourses. Matthew 16:26 answers `worth-of-the-soul-1` to `-6` (the test asserts it).

## How a row's page was found

`url` is `https://archive.org/details/<id>/page/n<N>`. N is the scan leaf counted from 0 (the number in the OCR file's page name, less one), so it counts the few leaves (colour cards, covers) the OCR leaves out; the Archive's own full-text search (`/fulltext/inside.php`) gives the same page numbers for the title lines it was asked about (Disputes Reviewed, Popery, the Fire of London, Friendly Visits, Tallents). A sermon's page is the leaf where its title or text line stands; where the title sits at the foot of a leaf the next one may be the page a reader wants (noted on `serious-address`). `sermons check --urls 52` and a HEAD of every url answered 200 for all 52 (a 200 proves the item and leaf exist, not that the page is the sermon; open two or three).

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
- The ordination charges (Charges I to IV and the one the OCR reads as "CHARGE VL" in "Sermons and Charges", and the Exhortation at Mr. Clark's ordination): charges with no text, printed after the Numbers xv. 39 sermon. They are the part of the probe's "about 20 numbered manuscript sermons" that is not a row: 6 + 4 + 4 + the Numbers sermon = 15 rows, and the charges are the rest.
- The 1782 Select Sermons and the single sermons of 1704 to 1735 on the Archive (`bim_eighteenth-century_...`), and the 1979 Baker reprint (a lending copy): not used.

## Sweep: three more preachers on the Archive as scans with printed texts

Candidates for the add-a-source recipe (docs/sources.md), not built. Each was checked for the OCR route (`<id>_djvu.txt` present, `possible-copyright-status` NOT_IN_COPYRIGHT where the Archive gives one) and for a printed text under each sermon's title; none was read beyond the first few sermons.

| Preacher | Item | What it is | Printed text | Notes |
| --- | --- | --- | --- | --- |
| George Whitefield (1714-1770) | `eighteensermonsp00whit` | *Eighteen sermons preached by the late Rev. George Whitefield*, 1771 | `SERMON I.` then the title, then the text (`Revel. xxii. 21`, `Psalm xlvi. 1-6`, `3 Ep. John ver. 2`) | the OCR keeps the long s (`Minifter`), so titles need care; the six-volume *Works* (`worksofreverendg01whit` to `06whit`, 1771) hold more |
| Jonathan Edwards (1703-1758) | `worksofpresident08edwa` | *The works of President Edwards*, Worcester, 1808, vol. 8: "Sermons on various important subjects", numbered (Sermon XVI onward in the OCR) | `SERMON XVI.`, the title, then the text in capitals (`HEBREWS v. 12.`) | status NOT_IN_COPYRIGHT; other volumes of the set (`worksofpresident02edwa` to `07edwa`) hold treatises and more sermons |
| John Wesley (1703-1791) | `sermonsonseveral01wesl` | *Sermons on several occasions*, New York, J. & J. Harper, 1831, vol. 1 (the contents give each sermon's title and text: "Salvation by Faith; Ephesians ii. 8") | the contents table prints title and text on one line | NOT_IN_COPYRIGHT; `standardsermonsc01wesluoft` (a 1951 reprint) shows no status and is not recommended |
