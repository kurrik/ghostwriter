Ghostwriter
===========
Represents a lingering unhapiness with jekyll, and a stupid impulse to write it
all myself.

Status
------
Starting to actually be a thing!
I'll update once I'm using this for actual projects.
Currently building the example site under `testsite`.  Run:

    $ cd testsite
    $ go run ../*.go --watch

Note that the `--watch` flag will re-run Ghostwriter if it picks up filesystem
changes.  So editing a site is as easy as changing the source directory and
reloading the page in your browser.

You can serve the output with:

    $ go run ../*.go --watch --serve=:8080

Which will host the rendered site at http://localhost:8080.  The value you
pass to the `--server` flag will be forwarded to
http://golang.org/pkg/net/http/#Server so anything that function accepts
should work.

Problems
--------
"too many open files"
Seems to happen on OSX because the default file limit is low (256).

You can see how many files ghostwriter has open by running:

    lsof | awk '{ print $2; }' | uniq -c | sort -rn | head

The second column is the process ID.  Find the ID which matches your Ghostwriter
process.  You can raise the limit temporarily by running:

    ulimit -n 4096

before executing your ghostwriter -watch command.

Dependencies
------------
Make sure you have bazaar installed.  In Ubuntu:

    sudo apt-get install bzr

Run:

    go get launchpad.net/goyaml
    go get github.com/russross/blackfriday
    go get github.com/howeyc/fsnotify
    go get github.com/kurrik/fauxfile

The goyaml library will show some warnings when compiling but it appears to have
no effect.
