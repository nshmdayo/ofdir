# ofdir zsh integration
# Usage: eval "$(ofdir --init zsh)"

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

# zsh completion for bookmark names
_ofdir_complete() {
    local state
    _arguments '*:: :->args'
    case $state in
        args)
            if [[ "${words[2]}" == @* ]]; then
                local -a bookmarks
                bookmarks=($(command ofdir --list-bookmarks 2>/dev/null))
                compadd -P @ -- "${bookmarks[@]#@}"
            else
                _directories
            fi
            ;;
    esac
}
compdef _ofdir_complete ofdir
