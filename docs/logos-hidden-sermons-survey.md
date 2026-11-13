# Sermons hidden inside the Governor's Logos books (mw-qtu3zvz.117)

His question (Postern general, 2026-11-13): are there books in his Logos library with sermons in them that Logos does not type as a book of sermons, and can they show up as available sermons too?

**Yes.** Of the 4,707 resources in his library (`plans/lampas-logos-library-2026-10-08.csv`) that are not among the 88 typed as sermon books, this survey found sermons inside collected works (Edwards, Luther, Fuller, Fletcher, Strong), a book of classic sermons and a few single-address books. The five with the most sermons that print their text were indexed and imported (`data/logos-index/`, `sources/*-logos`: 244 rows). The rest are listed below, in his well-known-first order, as follow-up stories.

<!-- toc -->

- [How it was done, and what it did not cover](#how-it-was-done-and-what-it-did-not-cover)
- [Sermons found, in his order](#sermons-found-in-his-order)
- [The books he named](#the-books-he-named)
- [Follow-up stories, in his order](#follow-up-stories-in-his-order)

<!-- /toc -->

## How it was done, and what it did not cover

- Read through his saved Logos web session (the same one `bin/logos-sermons-index.mjs` uses), GET only, one request a second. Nothing was written to his Logos account, and no sermon text is kept anywhere: only a title, the passage, the cited passages and the `ref.ly` address.
- **Candidates:** the 1,948 resources whose type can hold a collection: `text.monograph` (1,319), `collected-work`, `lecture`, `letters`, `devotional`, `personal-journal`, biography and autobiography, church history, Bible study, festschrift, prayers, handbooks and the like. Not read: Bibles, commentaries, reverse interlinears, lexicons, grammars, ancient-manuscript texts, media and datasets (a sermon is not hidden there), and the journals and magazines. Eleven candidates answered "not authorised" (the Reader's Edition of *Discipleship* and ten courseware workbooks) and could not be read: `LLS:DISCIPLESHIP`, `LLS:CM382CAMPBELL`, `LLS:CM382CAMPBELLWKBK`, `LLS:ED281MCGEVER`, `LLS:ED281MCGEVERWKBK`, `LLS:ET101ALLEN`, `LLS:ET101ALLENWKBK`, `LLS:NT101COHICK`, `LLS:NT101COHICKWKBK`, `LLS:OT319BLOCK`, `LLS:OT319BLOCKWKBK`.
- **Top level of every candidate's table of contents** was read; **238 resources were read deeper** (every collected work; every book with a heading that names a sermon, homily, discourse, address, lecture, oration or message; a short list of books whose own title says so). A book whose contents list never says "sermon" and whose title does not either (a collection of chapters each on a text) could be missed; the 52-verse and devotional books that came up in a check for headings that are passages (`Holiness Day by Day`, `52 Bible Verses You Should Have in Your Heart`, the Life Guide Bible studies) are devotionals and studies, not sermons, and were left out.
- **A section counts** when its heading, or the heading of the part it sits in, names a sermon, homily, discourse, address, lecture or oration. **It is indexed** only when the book prints the passage it is preached on (in the heading line, or in the first paragraphs as a Scripture link); otherwise it is **left out**, and the index file's report names it.
- The counts below are headings. "With a text" is the number of those the passage was found for; "not checked" means the sermon headings were counted but the sections were not opened.

## Sermons found, in his order

| Resource | Title | Author | Sermon sections | With a text | Three example headings | Done |
| --- | --- | --- | ---: | ---: | --- | --- |
| `LLS:WRKJONEDWRDS2` | The Works of Jonathan Edwards, Volume 2 | Jonathan Edwards | 63 | 63 | "Sermon II: Sinners in the Hands of an Angry God"; "Sermon I: Acts 16:29, 30"; "Christian Charity: Or, the Duty of Charity to the Poor, Explained and Enforced" | imported: `edwards-works-2-logos` |
| `LLS:WRKJONEDWRDS1` | The Works of Jonathan Edwards, Volume 1 | Jonathan Edwards | 6 | 5 | "Discourse I: Justification by Faith Alone"; "Discourse III: Ruth’s Resolution"; "A Farewell Sermon Preached at the First Precinct at Northampton" (no text printed: left out) | imported: `edwards-works-1-logos` |
| `LLS:NNTTDLTHR04` | The Annotated Luther, Volume 4: Pastoral Writings | Martin Luther | 15 | 3 | "The First Sermon, 9 March 1522, Invocavit Sunday" (the eight Invocavit sermons name no single text: left out); "Sermon Preached at Erfurt on the Journey to Worms, John 20:19–20, 7 April 1521"; "The Gospel for the Early Christmas Service, Luke 2[:15–20]" | imported: `luther-annotated-4-logos` |
| `LLS:ANNOTLUTV1` | The Annotated Luther, Volume 1: The Roots of Reform | Martin Luther | 3 | 0 | "A Sermon on Indulgences and Grace, 1518"; "A Sermon on the Meditation of Christ’s Holy Passion, 1519"; "Sermon on the Sacrament of Penance, 1519" | none has a text: left out (topical sermons) |
| `LLS:ANNOTLUTV2` | The Annotated Luther, Volume 2: Word and Faith | Martin Luther | 1 | 0 | "Sermon on Two Kinds of Righteousness 1519" | no text printed: left out |
| `LLS:NNTTDLTHR05` | The Annotated Luther, Volume 5: Christian Life in the World | Martin Luther | 1 | 0 | "A Sermon on the Estate of Marriage Revised and Corrected by Dr. Martin Luther" | no text printed: left out |
| `LLS:FULLER01` | The Complete Works of Andrew Fuller, Volume 1: Memoirs, Sermons, Etc. | Andrew Fuller | 96 | 84 | "Sermon I: The Nature and Importance of Walking by Faith (2 Cor. 5:7)"; "Sermon IV: The Blessedness of the Dead Who Die in the Lord (Rev. 14:13)"; "Sermon XLII: The Choice of Moses (Heb. 11:24–26)" | imported: `fuller-works-1-logos` |
| `LLS:FULLER03` | The Complete Works of Andrew Fuller, Volume 3: Expositions—Miscellaneous | Andrew Fuller | 95 | 89 | "Discourse I: The Book in General, and the First Day’s Creation, Chap. 1:1–5"; "Discourse IV: The Fall of Man, Chap. 3:1–7"; "Discourse XLIV: Jacob Removes to Bethel" | imported: `fuller-works-3-logos` (expository discourses on Genesis and the Apocalypse, each on its chapter) |
| `LLS:9780802491671` | Sermon Classics by Great Preachers | (several) | 8 | not checked | "1. The Fire Sermon"; "2. Sinners in the Hands of an Angry God"; "3. The Most Wonderful Sentence Ever Written" | follow-up |
| `LLS:WORKSFLETCH04` | The Works of the Reverend John Fletcher, Volume 4 | John Fletcher | 43 | 10 (from the heading) | "Sermon III.—State of the Natural Man (1 Cor. 2:14)"; "Sermon IV.—Awake, Thou that Sleepest (Eph. 5:14)"; "The Substance of a Sermon Preached on the Occasion (Num. 16:30–34)" | follow-up: the other 33 (outlines, vindications, other pieces) must be opened to see which are sermons |
| `LLS:WORKSARM01` | The Works of Arminius, Volume 1 | Jacobus Arminius | 6 | not checked | "Oration I: The Priesthood of Christ"; "Oration II: The Object of Theology"; "Oration III: The Author and the End of Theology" | follow-up (academic orations; probably no preaching text) |
| `LLS:MNTBLSSNGLBTTDS` | The Mount of Blessing; or, Lectures on the Beatitudes | (not read) | 9 | 0 | "Lecture I: The Great Teacher"; "Lecture II: First Beatitude"; "Lecture III: Second Beatitude" | lectures on Matthew 5, no Scripture link in the opening: left out; a follow-up could take the passage from the heading |
| `LLS:MISCNIES_02` | Miscellanies, Volume II: Chiefly Theological | A. H. Strong | 13 | not checked | "1900: Loyalty"; "1901: Right Beginnings"; "1902: More to Follow" | follow-up (the yearly addresses to graduating classes; the same author as `strong-logos`) |
| `LLS:CHRISTCRTNETHMON` | Christ in Creation and Ethical Monism | A. H. Strong | 12 | not checked | "1888: “A Son of Exhortation”"; "1889: Looking Forward"; "1890: No Striving" | follow-up (the same addresses, an earlier printing) |
| `LLS:WORKSFLETCH01`, `LLS:WORKSFLETCH02`, `LLS:WORKSFLETCH03` | The Works of the Reverend John Fletcher, Volumes 1 to 3 | John Fletcher | 1, 2, 1 | not checked | "Salvation by the Covenant of Grace: A Discourse on …"; "Section XX: Address to Perfect Christians" | follow-up (one or two each) |
| `LLS:BROOKS06` | The Complete Works of Thomas Brooks, Volume 6 | Thomas Brooks | 1 | not checked | "A Sermon" | follow-up |
| `LLS:JEHVAHMAGNFD` | Jehovah Magnified: Addresses | (not read) | 1 | not checked | "Address to Young Men" | follow-up |

Also swept, with a heading hit, and **not** sermons on a text: Wesley's *Works* volumes 9 to 13 (letters, a "Calm Address", an "Address to the Clergy"), *The Riches of John Bunyan* (the *Last Sermon* is already `bunyan-logos`), the *Letters of John Calvin*, *Lectures to My Students* 1 to 4 and *Lectures on the History of Preaching* (lectures on the craft), *Jesus of Nazareth* and *Lectures upon the Philosophy of History* (lectures), the Southern Historical Society, Medford Historical Society and Rebellion Record volumes (addresses at memorials), *Life and Public Services of Charles Sumner* and Wendell Phillips's speeches (political orations), and many books about preaching (*Mastering Contemporary Preaching*, *Preaching Evangelistically*, and so on), whose chapter headings carry the word "sermon" or "preaching".

## The books he named

| Resource | Title | Sermon sections | Why |
| --- | --- | ---: | --- |
| `LLS:WRKJONEDWRDS1` | The Works of Jonathan Edwards, Volume 1 | 6 (5 with a text) | The five Discourses ("Justification by Faith Alone", "Pressing into the Kingdom of God", "Ruth’s Resolution", "The Justice of God in the Damnation of Sinners", "The Excellency of Christ") and the Farewell Sermon. The rest of the volume is treatises (*Freedom of the Will*, *Original Sin*, *Religious Affections*). |
| `LLS:WRKJONEDWRDS2` | The Works of Jonathan Edwards, Volume 2 | 63 (63 with a text) | The Sermons, the Miscellaneous Discourses, and the Seventeen Occasional Sermons; each is headed by or opens with its text. |
| `LLS:ANNOTLUTV1` | The Annotated Luther, Volume 1: The Roots of Reform | 3 (0 with a text) | Three sermons of 1518 and 1519; none is on one printed text. |
| `LLS:ANNOTLUTV2` | The Annotated Luther, Volume 2: Word and Faith | 1 (0 with a text) | *Sermon on Two Kinds of Righteousness*; no text printed in its heading. |
| `LLS:NNTTDLTHR03` | The Annotated Luther, Volume 3: Church and Sacraments | none found | Treatises (*The Babylonian Captivity of the Church*, *The German Mass*, *That These Words of Christ …*); no heading names a sermon. |
| `LLS:NNTTDLTHR04` | The Annotated Luther, Volume 4: Pastoral Writings | 15 (3 with a text) | "Sermon and Song", with the Invocavit Sermons (eight, no single text), Selected Sermons and Selected Hymns; three headings print a text. |
| `LLS:NNTTDLTHR05` | The Annotated Luther, Volume 5: Christian Life in the World | 1 (0 with a text) | *A Sermon on the Estate of Marriage*; no text printed. |
| `LLS:NNTTDLTHR06` | The Annotated Luther, Volume 6: The Interpretation of Scripture | none found | Prefaces to the books of the Bible and the *Lectures on Genesis* (an academic commentary); the headings that say "lecture" are not sermons preached on a text. |
| `LLS:BONHOEFFERREADER` | The Bonhoeffer Reader | none found | Forty-four extracts: lectures (the Inaugural Lecture, *Lecture on “War”*, *Lectures on Christology*), a Fanø address on peace, theology and ethics, letters from prison. No heading names a sermon preached on a text; the "Sermon on the Mount" chapter is a chapter of *Discipleship*. |

## Follow-up stories, in his order

1. **Luther volumes 1, 2 and 5:** five sermons with no printed text; take the passage from the sermon's own heading lines or its marginal Scripture links, or leave them out for good. (Not before the Governor says he wants Luther's topical sermons listed with no passage.)
2. **`LLS:9780802491671` Sermon Classics by Great Preachers:** eight classic sermons; read each for its text; import.
3. **`LLS:WORKSFLETCH04` Fletcher, Volume 4:** open the 43 sections, keep the sermons, import (10 already name their text in the heading); then Fletcher's volumes 1 to 3.
4. **`LLS:WORKSARM01` Arminius's Works, Volume 1:** six orations; probably no preaching text, check first.
5. **`LLS:MISCNIES_02` and `LLS:CHRISTCRTNETHMON`, A. H. Strong:** the 1888 to 1900 addresses (one set, two printings); check for a text, and avoid importing the same address twice (`--dedupe-titles`).
6. **`LLS:MNTBLSSNGLBTTDS`, `LLS:BROOKS06`, `LLS:JEHVAHMAGNFD`:** one decision each; the lectures on the Beatitudes are on Matthew 5:3–12 but print no link.
7. **The eleven books that answered "not authorised"** (above) if any of them is a sermon book; the library CSV types them as courseware and a Reader's Edition, so none is expected.
