/^[^-][\/a-zA-Z\-_0-9:\\]+:(\s.+)?$/ {
    split($1,A,/ /);

    helpCommand = A[1];
    gsub(/\\/, "", helpCommand);  # unescape
    gsub(/:$/, "", helpCommand);
    print helpCommand;
}
