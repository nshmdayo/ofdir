# opfd zsh integration
# Usage: eval "$(opfd --init zsh)"

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

# zsh completion for bookmark names
_opfd_complete() {
    local state
    _arguments '*:: :->args'
    case $state in
        args)
            if [[ "${words[2]}" == @* ]]; then
                local -a bookmarks
                bookmarks=($(command opfd --list-bookmarks 2>/dev/null))
                compadd -P @ -- "${bookmarks[@]#@}"
            else
                _directories
            fi
            ;;
    esac
}
compdef _opfd_complete opfd
