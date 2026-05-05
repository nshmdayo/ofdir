# ofdir bash integration
# Usage: eval "$(ofdir --init bash)"

function ofdir() {
    # No arguments: go home
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    # Capture ofdir output; UI and errors go to the terminal via /dev/tty
    local target
    target=$(command ofdir "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        if builtin cd "$target"; then
            command ofdir --record "$target" &>/dev/null &
        fi
    fi
    return $exit_code
}

# Tab completion for bookmark names
_ofdir_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == @* ]]; then
        local names
        names=$(ofdir --list-bookmarks 2>/dev/null)
        COMPREPLY=($(compgen -W "$names" -- "$cur"))
    else
        # Fall back to directory completion
        COMPREPLY=($(compgen -d -- "$cur"))
    fi
}
complete -F _ofdir_completion ofdir
