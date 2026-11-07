# worlds-great-sermons-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-1-basil-to-calvin-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-2-hooker-to-south-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-3-massillon-to-mason-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-4-l-beecher-to-bushnell-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-5-guthrie-to-mozley-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-6-h-w-beecher-to-punshon-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-7-hale-to-farrar-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-8-talmage-to-knox-little-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-9-cuyler-to-van-dyke-sermons.jsonl --in <vault>/plans/sermons-2026-11-05/the-world-s-great-sermons-volume-10-drummond-to-jowett-sermons.jsonl --title-form "preacher-years" --id worlds-great-sermons-logos --out sources/worlds-great-sermons-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          114
rows written:                        113
lines dropped:                       0
non-sermon lines dropped:            1
rows with a ref:                     0
  of them read from the label:       0
  of them with cited references:     110
rows with no ref:                    113
series rows (ref over >3 chapters):  0
rows with their own date:            0
rows dated by the year alone:        0
references the book table refused:   1
Logos and the label disagree:        0
```

## Rows before and after, by input

| input | lines | rows before | duplicates dropped | rows after |
|---|---|---|---|---|
| the-world-s-great-sermons-volume-1-basil-to-calvin | 10 | 10 | 0 | 10 |
| the-world-s-great-sermons-volume-2-hooker-to-south | 10 | 10 | 0 | 10 |
| the-world-s-great-sermons-volume-3-massillon-to-mason | 11 | 11 | 0 | 11 |
| the-world-s-great-sermons-volume-4-l-beecher-to-bushnell | 11 | 11 | 0 | 11 |
| the-world-s-great-sermons-volume-5-guthrie-to-mozley | 9 | 9 | 0 | 9 |
| the-world-s-great-sermons-volume-6-h-w-beecher-to-punshon | 11 | 11 | 0 | 11 |
| the-world-s-great-sermons-volume-7-hale-to-farrar | 13 | 13 | 0 | 13 |
| the-world-s-great-sermons-volume-8-talmage-to-knox-little | 13 | 13 | 0 | 13 |
| the-world-s-great-sermons-volume-9-cuyler-to-van-dyke | 14 | 14 | 0 | 14 |
| the-world-s-great-sermons-volume-10-drummond-to-jowett | 12 | 11 | 0 | 11 |
| **all** | 114 | 113 | 0 | 113 |

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## Non-sermon lines dropped (1)

The archive's index lists them with the sermons, but they are a table, an index of Scripture, an introduction or a book's argument (prolegomena), not a sermon: not written, and not counted in "rows before".

- WRLDGRTSERM10.1-273085: "General Index" (an index)

## Titles that did not have the form (0)

--title-form "preacher-years" reads the preacher, years, text or number printed in a title; these titles do not have it, so the row keeps the title and the preacher as the index gives them.

None.

## References the book table refused (1)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

- WRLDGRTSERM01.1-32517: "bible.44.22.10": Logos book 44 is not one of the 66

## Series rows (0)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

None.

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Rows with no ref (113)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

WRLDGRTSERM01.1-5589, WRLDGRTSERM01.1-32517, WRLDGRTSERM01.1-61034, WRLDGRTSERM01.1-92325, WRLDGRTSERM01.1-113333, WRLDGRTSERM01.1-139961, WRLDGRTSERM01.1-178832, WRLDGRTSERM01.1-194016, WRLDGRTSERM01.1-207039, WRLDGRTSERM01.1-245673, WRLDGRTSERM02.1-1004, WRLDGRTSERM02.1-35706, WRLDGRTSERM02.1-71234, WRLDGRTSERM02.1-102368, WRLDGRTSERM02.1-124829, WRLDGRTSERM02.1-162721, WRLDGRTSERM02.1-189337, WRLDGRTSERM02.1-217009, WRLDGRTSERM02.1-242524, WRLDGRTSERM02.1-262880, WRLDGRTSERM03.1-993, WRLDGRTSERM03.1-25986, WRLDGRTSERM03.1-56653, WRLDGRTSERM03.1-83409, WRLDGRTSERM03.1-106527, WRLDGRTSERM03.1-128409, WRLDGRTSERM03.1-155911, WRLDGRTSERM03.1-186078, WRLDGRTSERM03.1-209178, WRLDGRTSERM03.1-229629, WRLDGRTSERM03.1-252455, WRLDGRTSERM04.1-1481, WRLDGRTSERM04.1-31482, WRLDGRTSERM04.1-63441, WRLDGRTSERM04.1-96815, WRLDGRTSERM04.1-122596, WRLDGRTSERM04.1-159027, WRLDGRTSERM04.1-172537, WRLDGRTSERM04.1-203897, WRLDGRTSERM04.1-225783, WRLDGRTSERM04.1-246853, WRLDGRTSERM04.1-277606, WRLDGRTSERM05.1-1138, WRLDGRTSERM05.1-27704, WRLDGRTSERM05.1-54291, WRLDGRTSERM05.1-73074, WRLDGRTSERM05.1-104513, WRLDGRTSERM05.1-142662, WRLDGRTSERM05.1-175536, WRLDGRTSERM05.1-214337, WRLDGRTSERM05.1-248013, WRLDGRTSERM06.1-1599, WRLDGRTSERM06.1-32663, WRLDGRTSERM06.1-62995, WRLDGRTSERM06.1-81780, WRLDGRTSERM06.1-102682, WRLDGRTSERM06.1-133208, WRLDGRTSERM06.1-156633, WRLDGRTSERM06.1-173264, WRLDGRTSERM06.1-196093, WRLDGRTSERM06.1-229734, WRLDGRTSERM06.1-259782, WRLDGRTSERM07.1-1743, WRLDGRTSERM07.1-22800, WRLDGRTSERM07.1-47460, WRLDGRTSERM07.1-76322, WRLDGRTSERM07.1-104607, WRLDGRTSERM07.1-120984, WRLDGRTSERM07.1-142413, WRLDGRTSERM07.1-168718, WRLDGRTSERM07.1-192272, WRLDGRTSERM07.1-210652, WRLDGRTSERM07.1-231756, WRLDGRTSERM07.1-244448, WRLDGRTSERM07.1-265745, WRLDGRTSERM08.1-1748, WRLDGRTSERM08.1-21631, WRLDGRTSERM08.1-47749, WRLDGRTSERM08.1-70475, WRLDGRTSERM08.1-92770, WRLDGRTSERM08.1-119576, WRLDGRTSERM08.1-138821, WRLDGRTSERM08.1-151206, WRLDGRTSERM08.1-168888, WRLDGRTSERM08.1-190410, WRLDGRTSERM08.1-203511, WRLDGRTSERM08.1-233214, WRLDGRTSERM08.1-253375, WRLDGRTSERM09.1-1874, WRLDGRTSERM09.1-22597, WRLDGRTSERM09.1-41774, WRLDGRTSERM09.1-54576, WRLDGRTSERM09.1-82896, WRLDGRTSERM09.1-101987, WRLDGRTSERM09.1-124626, WRLDGRTSERM09.1-144580, WRLDGRTSERM09.1-161288, WRLDGRTSERM09.1-184461, WRLDGRTSERM09.1-203329, WRLDGRTSERM09.1-220542, WRLDGRTSERM09.1-232958, WRLDGRTSERM09.1-256430, WRLDGRTSERM10.1-1788, WRLDGRTSERM10.1-46019, WRLDGRTSERM10.1-70399, WRLDGRTSERM10.1-96138, WRLDGRTSERM10.1-114068, WRLDGRTSERM10.1-128988, WRLDGRTSERM10.1-162839, WRLDGRTSERM10.1-184739, WRLDGRTSERM10.1-220863, WRLDGRTSERM10.1-237046, WRLDGRTSERM10.1-251961

## No row has a Scripture ref

None of the 113 rows has a Scripture ref (the archive's index names no text for any of them; any dates and series it has are kept, and the 110 that cite passages cite them without taking one as the text): none can show on a verse, and all are found by title, preacher and source. A row the index dates by no day is listed here.

### Rows with no date of their own (113)

WRLDGRTSERM01.1-5589, WRLDGRTSERM01.1-32517, WRLDGRTSERM01.1-61034, WRLDGRTSERM01.1-92325, WRLDGRTSERM01.1-113333, WRLDGRTSERM01.1-139961, WRLDGRTSERM01.1-178832, WRLDGRTSERM01.1-194016, WRLDGRTSERM01.1-207039, WRLDGRTSERM01.1-245673, WRLDGRTSERM02.1-1004, WRLDGRTSERM02.1-35706, WRLDGRTSERM02.1-71234, WRLDGRTSERM02.1-102368, WRLDGRTSERM02.1-124829, WRLDGRTSERM02.1-162721, WRLDGRTSERM02.1-189337, WRLDGRTSERM02.1-217009, WRLDGRTSERM02.1-242524, WRLDGRTSERM02.1-262880, WRLDGRTSERM03.1-993, WRLDGRTSERM03.1-25986, WRLDGRTSERM03.1-56653, WRLDGRTSERM03.1-83409, WRLDGRTSERM03.1-106527, WRLDGRTSERM03.1-128409, WRLDGRTSERM03.1-155911, WRLDGRTSERM03.1-186078, WRLDGRTSERM03.1-209178, WRLDGRTSERM03.1-229629, WRLDGRTSERM03.1-252455, WRLDGRTSERM04.1-1481, WRLDGRTSERM04.1-31482, WRLDGRTSERM04.1-63441, WRLDGRTSERM04.1-96815, WRLDGRTSERM04.1-122596, WRLDGRTSERM04.1-159027, WRLDGRTSERM04.1-172537, WRLDGRTSERM04.1-203897, WRLDGRTSERM04.1-225783, WRLDGRTSERM04.1-246853, WRLDGRTSERM04.1-277606, WRLDGRTSERM05.1-1138, WRLDGRTSERM05.1-27704, WRLDGRTSERM05.1-54291, WRLDGRTSERM05.1-73074, WRLDGRTSERM05.1-104513, WRLDGRTSERM05.1-142662, WRLDGRTSERM05.1-175536, WRLDGRTSERM05.1-214337, WRLDGRTSERM05.1-248013, WRLDGRTSERM06.1-1599, WRLDGRTSERM06.1-32663, WRLDGRTSERM06.1-62995, WRLDGRTSERM06.1-81780, WRLDGRTSERM06.1-102682, WRLDGRTSERM06.1-133208, WRLDGRTSERM06.1-156633, WRLDGRTSERM06.1-173264, WRLDGRTSERM06.1-196093, WRLDGRTSERM06.1-229734, WRLDGRTSERM06.1-259782, WRLDGRTSERM07.1-1743, WRLDGRTSERM07.1-22800, WRLDGRTSERM07.1-47460, WRLDGRTSERM07.1-76322, WRLDGRTSERM07.1-104607, WRLDGRTSERM07.1-120984, WRLDGRTSERM07.1-142413, WRLDGRTSERM07.1-168718, WRLDGRTSERM07.1-192272, WRLDGRTSERM07.1-210652, WRLDGRTSERM07.1-231756, WRLDGRTSERM07.1-244448, WRLDGRTSERM07.1-265745, WRLDGRTSERM08.1-1748, WRLDGRTSERM08.1-21631, WRLDGRTSERM08.1-47749, WRLDGRTSERM08.1-70475, WRLDGRTSERM08.1-92770, WRLDGRTSERM08.1-119576, WRLDGRTSERM08.1-138821, WRLDGRTSERM08.1-151206, WRLDGRTSERM08.1-168888, WRLDGRTSERM08.1-190410, WRLDGRTSERM08.1-203511, WRLDGRTSERM08.1-233214, WRLDGRTSERM08.1-253375, WRLDGRTSERM09.1-1874, WRLDGRTSERM09.1-22597, WRLDGRTSERM09.1-41774, WRLDGRTSERM09.1-54576, WRLDGRTSERM09.1-82896, WRLDGRTSERM09.1-101987, WRLDGRTSERM09.1-124626, WRLDGRTSERM09.1-144580, WRLDGRTSERM09.1-161288, WRLDGRTSERM09.1-184461, WRLDGRTSERM09.1-203329, WRLDGRTSERM09.1-220542, WRLDGRTSERM09.1-232958, WRLDGRTSERM09.1-256430, WRLDGRTSERM10.1-1788, WRLDGRTSERM10.1-46019, WRLDGRTSERM10.1-70399, WRLDGRTSERM10.1-96138, WRLDGRTSERM10.1-114068, WRLDGRTSERM10.1-128988, WRLDGRTSERM10.1-162839, WRLDGRTSERM10.1-184739, WRLDGRTSERM10.1-220863, WRLDGRTSERM10.1-237046, WRLDGRTSERM10.1-251961
