# opfd bash integration
# Usage: eval "$(opfd --init bash)"

function opfd() {
    # No arguments: go home
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    # Capture opfd output; UI and errors go to the terminal via /dev/tty
    local target
    target=$(command opfd "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        if builtin cd "$target"; then
            command opfd --record "$target" &>/dev/null &
        fi
    fi
    return $exit_code
}

# Tab completion for bookmark names
_opfd_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == @* ]]; then
        local names
        names=$(command opfd --list-bookmarks 2>/dev/null)
        COMPREPLY=($(compgen -W "$names" -- "$cur"))
    else
        # Fall back to directory completion
        COMPREPLY=($(compgen -d -- "$cur"))
    fi
}
complete -F _opfd_completion opfd
