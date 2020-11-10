# pagexmlcat
Concatenate
[PageXML](http://www.primaresearch.org/publications/ICPR2010_Pletschacher_PAGE)-formatted
xml files.

## Usage
`pagexmlcat [OPTIONS] [FILES...]`

Concatenate FILE(s) to standard output.  With no file or if file is
`-`, read standard input.

## Options
`-h` print help

`-index` comma-separated list of (zero-based) indices to select from
multiple TextEquiv elements (negative indices count from the end)

`-serial` ignore region ordering in the document and use the explicit
region ordering of the document

`-id` prefix output lines with their respective line (or word) ids

`-conf` prefix output lines with their respective confidences (if
available)

`-region` set region type to output (line|word|glyph|block); default
is line

`-filename` output the filename of printed regions

`-norm` replace each space with `_` in output text

## Examples
`pagexmlcat a.xml - b.xml` Output a.xml's contents, then standard
input, then b.xml's contents.

`pagexmlcat` Output document from standard input to standard output.

`pagexmlcat -index 0,-1` Output the first and last text equiv region
for each line from standard input to standard output.

`pagexmlcat -region word a.xml` Output a.xml's words to standard
output.

`pagexmlcat -region block a.xml` Output a.xml's text regions to
standard output.

## Author
Written by Florian Fink
