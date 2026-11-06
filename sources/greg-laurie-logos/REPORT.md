# greg-laurie-logos: what the import did not settle

<!-- toc: none -->

Written by `go run ./tools/logos-import --in <vault>/plans/sermons-2026-11-05/greg-laurie-sermons.jsonl --id greg-laurie-logos --out sources/greg-laurie-logos`; do not edit by hand. Only ids, titles, references and addresses are listed.

## Counts

```
lines read:                          1196
rows written:                        1196
lines dropped:                       0
rows with a ref:                     1091
  of them read from the label:       14
  of them with cited references:     41
rows with no ref:                    105
series rows (ref over >3 chapters):  25
rows with their own date:            632
rows dated by the year alone:        564
references the book table refused:   0
Logos and the label disagree:        0
```

## Lines dropped (0)

A line with neither a ref nor a title, or with no id, no title, no address or an id already used, is not a row.

None.

## References the book table refused (0)

A Logos reference to a book outside the sixty-six, or a chapter or verse past the end; it is left out of the row, and the row stays.

None.

## Series rows (25)

The ref spans more than three chapters (a whole book or a long text): the row is kept, found by its title and listed by its source, and matched by no passage.

- YEAR.1996.17 Jonah (4 chapters)
- YEAR.1996.18 Jonah (4 chapters)
- YEAR.1998.5 Jas (5 chapters)
- YEAR.2000.18 1Pet (5 chapters)
- YEAR.2002.49 Judg (21 chapters)
- YEAR.2003.13 Gen.4-Gen.11 (8 chapters)
- YEAR.2003.14 Gen.11-Gen.18 (8 chapters)
- YEAR.2003.15 Gen.16-Gen.20 (5 chapters)
- YEAR.2003.16 Gen.27-Gen.32 (6 chapters)
- YEAR.2003.17 Gen.33-Gen.36 (4 chapters)
- YEAR.2003.18 Gen.37-Gen.41 (5 chapters)
- YEAR.2003.31 Exod.1-Exod.5 (5 chapters)
- YEAR.2004.12 Num.1-Num.9 (9 chapters)
- YEAR.2004.47 Judg.1-Judg.5 (5 chapters)
- YEAR.2004.51 Judg.13-Judg.16 (4 chapters)
- YEAR.2006.20 Exod.5-Exod.10 (6 chapters)
- YEAR.2007.3 Eccl (12 chapters)
- YEAR.2007.12 Esth (10 chapters)
- YEAR.2007.13 Esth (10 chapters)
- YEAR.2007.22 Jonah (4 chapters)
- YEAR.2009.29 Eccl (12 chapters)
- YEAR.2012.48 Gen.37-Gen.45 (9 chapters)
- YEAR.2012.55 Exod.2-Exod.6 (5 chapters)
- YEAR.2012.56 Jonah (4 chapters)
- YEAR.2012.57 Exod.5-Exod.10 (6 chapters)

## Logos and the label disagree (0)

Both were read and differ; the row carries Logos's reference.

None.

## Rows with no ref (105)

Neither a Logos reference nor a label that could be read: the row is kept, found by its title and its source, and by no passage. Ids, in the file's order:

YEAR.1994.20, YEAR.1997.6, YEAR.1997.7, YEAR.1997.15, YEAR.1997.30, YEAR.1999.10, YEAR.2000.2, YEAR.2000.44, YEAR.2001.36, YEAR.2001.37, YEAR.2001.38, YEAR.2001.39, YEAR.2001.40, YEAR.2001.41, YEAR.2001.44, YEAR.2001.47, YEAR.2001.48, YEAR.2001.49, YEAR.2001.55, YEAR.2001.58, YEAR.2001.59, YEAR.2001.60, YEAR.2001.62, YEAR.2001.65, YEAR.2001.66, YEAR.2001.68, YEAR.2001.69, YEAR.2001.70, YEAR.2001.71, YEAR.2001.72, YEAR.2001.73, YEAR.2001.77, YEAR.2001.79, YEAR.2001.80, YEAR.2002.1, YEAR.2002.4, YEAR.2002.6, YEAR.2002.12, YEAR.2002.21, YEAR.2002.22, YEAR.2002.23, YEAR.2002.26, YEAR.2002.31, YEAR.2002.32, YEAR.2002.67, YEAR.2002.79, YEAR.2003.1, YEAR.2003.3, YEAR.2003.4, YEAR.2003.22, YEAR.2003.29, YEAR.2003.33, YEAR.2003.62, YEAR.2003.74, YEAR.2003.75, YEAR.2004.16, YEAR.2004.21, YEAR.2004.22, YEAR.2004.23, YEAR.2004.24, YEAR.2004.25, YEAR.2004.26, YEAR.2004.27, YEAR.2004.36, YEAR.2004.37, YEAR.2004.81, YEAR.2005.14, YEAR.2006.14, YEAR.2006.34, YEAR.2006.61, YEAR.2007.36, YEAR.2008.4, YEAR.2008.21, YEAR.2008.52, YEAR.2008.53, YEAR.2008.54, YEAR.2009.31, YEAR.2009.51, YEAR.2010.8, YEAR.2010.9, YEAR.2010.10, YEAR.2010.17, YEAR.2010.18, YEAR.2010.26, YEAR.2010.29, YEAR.2010.30, YEAR.2010.32, YEAR.2010.34, YEAR.2010.37, YEAR.2010.46, YEAR.2010.47, YEAR.2010.48, YEAR.2010.54, YEAR.2010.57, YEAR.2011.51, YEAR.2011.68, YEAR.2011.69, YEAR.2011.79, YEAR.2011.80, YEAR.2012.2, YEAR.2012.14, YEAR.2012.73, YEAR.2013.17, YEAR.2013.29, YEAR.2013.33
