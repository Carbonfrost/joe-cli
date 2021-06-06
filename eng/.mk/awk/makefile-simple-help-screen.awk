# Can work with downlevel AWK versions

# Makefile (phony) targets are documented using two ## right before its name
# e.g.:
# ## Help documentation
# help: ...
#
# This AWK script will parse them out and display
function color(code) {
    return sprintf("%c[%dm", 27, code);
}

/^[^-][\/a-zA-Z\-\_0-9]+:/ {
    helpMessage = match(lastLine, /(.+)/);
    _CYAN = color(34);
    _RESET = color(0);

    if (helpMessage) {
        helpCommand = substr($1, 0, index($1, ":")-1);
        helpMessage = lastLine;
        gsub(/##/, " ", helpMessage);
        printf "  %s%-16s%s %s\n", _CYAN, helpCommand, _RESET, helpMessage;
        lastLine = "";
    }
}
{
    if (match($0, /^##/)) {
        lastLine = lastLine $0
    }
}
