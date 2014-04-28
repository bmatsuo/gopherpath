Do not use this repository. It's far too immature.

Gopherpath -- reliably hosted Go import paths

Gopherpath is an application for Google App Engine and serves import metadata
for the `go get` tool.  It lets you easily 'self-host' your own import paths
(e.g. go.bmats.co/gopherpath/importmeta) in an environment where you don't have
to actually worry about maintaining servers.

The tool is similar in spirit to [gopkg.in](gopkg.in), and may eventually pull
in some of its features (semantic version support).  The primary difference is
that gopherpath only serves metadata for a single github user (you).

Ideally there will be a script to automate as much of the setup as possible.
For now, read the guide on [Getting Started](tree/master/GettingStarted.markdown).


Contributing


Create issues before submitting pull requests.  I love outside contribution.
But I'd like for contribution to be fairly structured.  This code will be
deployed in 'production' by many people (hopefully ;]).  Unregulated
contribution and unstable code puts your own `go get` calls at risk of failing!
