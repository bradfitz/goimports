This tool updates your Go import lines, adding missing ones and
removing unreferenced ones.

     $ go get golang.org/x/tools/cmd/goimports

     Note the new location. This project has moved to the official
     go.tools repo. Pull requests here will no longer be accepted.
     Please use the Go process: http://golang.org/doc/contribute.html

It acts the same as gofmt (same flags, etc) but in addition to code
formatting, also fixes imports.

See usage and editor integration notes, now moved elsewhere:

   http://godoc.org/golang.org/x/tools/cmd/goimports
   
This specific fork of [bradfitz goimports](https://github.com/bradfitz/goimports), cleans all blank lines in imports block before formatting

So, usually when an import is added by the IDE it adds it at the top of the dependency block. Like this:
```
import (
    "github.com/guyincogninto/costumes"
    "bytes"
    "flag"
    "fmt"

    "github.com/elbarto/graffiti"
)
```

Current go imports `-w` option formats and leave it like this:

```
import (
    "bytes"
    "flag"
    "fmt"

    "github.com/guyincogninto/costumes"

    "github.com/elbarto/graffiti"
)
```

Maybe I'm a bit fussy, but it's not desirable for me. This fork will leave the thing like this:
```
import (
    "bytes"
    "flag"
    "fmt"

    "github.com/elbarto/graffiti"
    "github.com/guyincogninto/costumes"
)
```

Happy hacking!
