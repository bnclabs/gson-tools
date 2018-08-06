README
======

[![IRC #bnc](https://www.irccloud.com/invite-svg?channel=%23bnc&amp;hostname=irc.mozilla.org&amp;port=6667)](https://www.irccloud.com/invite?channel=%23bnc&amp;hostname=irc.mozilla.org&amp;port=6667)

Collection of tools for Gson quality control.

* `validate/` sub-package is a standalone program to thrash all gson
  transformations with random set of data.
* `collate_validate/` sub-package is a standalone program to thrash
  collation algorithm with different set of configurations.
* `testdata/` is data directory for validate/ and collate_validate/.

```go
$ cd validate
$ ./check.sh
```

```go
$ cd collate_validate
$ ./check.sh
```
