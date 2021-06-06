# Requires GNU Awk, not just plain AWK.

# Makefile (phony) targets are documented using two ## right before its name
# e.g.:
# ## Help documentation
# help: ...
#
# This AWK script will parse them out and display
function color(code) {
    return sprintf("%c[%dm", 27, code);
}

function push(A,B) {
    A[length(A)+1] = B
}

function rindex(str, search, pos, res) {
    do {
        res = index(substr(str, pos + 1), search);
        pos += res;
    } while (res >= 1);
    return pos;
}

/^#:(.+)/ {
    group_description[FILENAME] = substr($$1, 4)
}
# Targets, except private targets leading with -
/^[^-][\/a-zA-Z\-_0-9:\\]+:(\s.+)?$/ {
    helpMessage = match(lastLine, /(.+)/);
    _CYAN = color(34)
    _RESET = color(0)

    if (helpMessage) {
        helpCommand = substr($$1, 0, rindex($$1, ":")-1);
        gsub(/\\/, "", helpCommand);  # unescape
        helpMessage = lastLine;
        gsub(/##/, " ", helpMessage);

        # Store the help message grouped by its file name
        groups[FILENAME][0] = ""
        push(groups[FILENAME], sprintf("  %s%-18s%s %s\n", _CYAN, helpCommand, _RESET, helpMessage));

        lastLine = "";
    }
}
{
    if (match($$0, /^##/)) {
        lastLine = lastLine $$0
    }
}
END{
    n = asorti(groups, sorted)
    for (i = 1; i <= n; i++) {
        a = sorted[i]
        if (group_description[a]) {
            print "";
        }
        print group_description[a];

        asort(groups[a], items);
        for (x = 1; x <= length(items); x++) {
            printf items[x]
        }
    }
}
