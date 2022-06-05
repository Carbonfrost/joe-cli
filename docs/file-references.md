# File references

It is sometimes convenient to use a flag or arg to name a file but use the file contents as the actual value.  In  Joe-cli, there are a few ways to accomplish this.

## `FilePath`

`Flag` and `Arg` have a field named `FilePath` which allows loading a file.  When used, this indicates that the given file will be loaded and its text or bytes will be used as the actual value.  If the file does not exist, this is not treated as an error.

## `FileReference` and `AllowFileReference`

You can use the options `FileReference` or `AllowFileReference` to interpret the text of the flag or arg as the name of a file which is transparently loaded and passed to the actual value.  The difference between the two options is that with `AllowFileReference`, whether the text is interpreted as a file or not depends upon whether it starts with an at sign (`@`).

The file specified by a file reference must exist.

## Example

Here's a small app that demonstrates the use of `FilePath` and `FileReference`.

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"

    "github.com/Carbonfrost/joe-cli"
)

func main() {
    (&cli.App{
        Name: "hmac",
        Flags: []*cli.Flag{
            {
                Name:     "key",
                Aliases:  []string{"k"},
                Value:    new([]byte),
                Options:  cli.FileReference,
                FilePath: os.ExpandEnv("$HOME/key.bin"),
            },
        },
        Args: []*cli.Arg{
            {
                Name:  "file",
                Value: new(cli.File),
                NArg:  1,
            },
        },
        Action: signFile,
    }).Run(os.Args)
}

func signFile(c *cli.Context) error {
    key := c.Bytes("key")
    mac := hmac.New(sha256.New, key)
    f, err := c.File("file").Open()
    if err != nil {
        return err
    }
    if _, err := io.Copy(mac, f); err != nil {
        return err
    }

    c.Stdout.WriteString(hex.EncodeToString(mac.Sum(nil)))
    c.Stdout.WriteString("\n")
    return nil
}

```

If you had a default key stored in your home directory, that key will be loaded by `FilePath`.  You can however specify your own key as a flag:

```sh
# create home key and a second test key
head -c 32 /dev/urandom > $HOME/key.bin
head -c 32 /dev/urandom > example.bin

# sign file using the key from file path
./hmac message.txt

# sign file using the specified file
./hmac -k example.bin message.txt

```
