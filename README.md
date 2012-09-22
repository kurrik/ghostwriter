Ghostwriter
===========
Represents a lingering unhapiness with jekyll, and a stupid impulse to write it
all myself.

Status
------
Still really rough.  I'll update once I'm using this for actual projects.
Currently building the example site under `testsite`.  Run:

    $ cd testsite
    $ go run ../main.go

You can serve the output with:

    $ cd testsite/dst
    $ python -m SimpleHTTPServer

Which will host the rendered site at http://localhost:8000
Note that you'll need to re-run Ghostwriter if you make source changes.

Dependencies
------------
Make sure you have bazaar installed.  In Ubuntu:

    sudo apt-get install bzr

Run:

    go get launchpad.net/goyaml
    go get github.com/knieriem/markdown
    go get github.com/kurrik/fauxfile

The goyaml library will show some warnings when compiling but it appears to have
no effect.
