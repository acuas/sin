#!/usr/bin/env bash

# Examples:
#     sin hello.txt world.py     # paste files.
#     echo Hello world. | sin    # read from STDIN.
#     sin                        # Paste in terminal.

sin() {
    local sin_HOST=http://localhost:8081
    [ -t 0 ] && {

        [ $# -gt 0 ] && {
            for filename in "$@"
            do
                if [ -f "$filename" ]
                then
                    curl -F f:1=@"$filename" $sin_HOST
                else
                    echo "file '$filename' does not exist!"
                fi
            done
            return
        }

        echo "^C to cancel, ^D to send."
    }
    curl -F f:1='<-' $sin_HOST
}

sin $*
