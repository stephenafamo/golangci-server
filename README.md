# golangci-server

golangci-server is a [golangci-lint](https://github.com/golangci/golangci-lint) language server.

## Installation

1. Install [golangci-lint](https://golangci-lint.run)
2. Install `golangci-server`

        go install github.com/stephenafamo/golangci-server@latest

## Inspiration

Most of the inspiration for this was taken from [golangci-lint-langserver](https://github.com/nametake/golangci-lint-langserver).

I started out trying to modify it but found it easier to just rebuild based on the excellent [glsp package](https://github.com/tliron/glsp)

### Configuration for [coc.nvim](https://github.com/neoclide/coc.nvim)

coc-settings.json

```json
{
  "languageserver": {
    "golangci-server": {
      "command": "golangci-server",
      "filetypes": ["go"],
    }
  }
}
```

### Configuration for [vim-lsp](https://github.com/prabirshrestha/vim-lsp)

```vim
augroup vim_lsp_golangci_server
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'golangci-server',
      \ 'cmd': {server_info->['golangci-server']},
      \ 'whitelist': ['go'],
      \ })
augroup END
```

### Configuration for [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)

```lua
local lspconfig = require 'lspconfig'
local configs = require 'lspconfig.configs'

if not configs.golangcilsp then
 	configs.golangcilsp = {
		default_config = {
			cmd = {'golangci-server'},
			root_dir = lspconfig.util.root_pattern('.git', 'go.mod'),
		};
	}
end
lspconfig.golangcilsp.setup {
	filetypes = {'go'}
}
```

### Configuration for [lsp-mode](https://github.com/emacs-lsp/lsp-mode) (Emacs)

```emacs-lisp
(with-eval-after-load 'lsp-mode
  (lsp-register-client
   (make-lsp-client :new-connection (lsp-stdio-connection
                                     '("golangci-server"))
                    :major-modes '(go-mode)
                    :language-id "go"
                    :priority 0
                    :server-id 'golangci-lint
                    :add-on? t
                    :library-folders-fn #'lsp-go--library-default-directories

  (add-to-list 'lsp-language-id-configuration '(go-mode . "golangci-lint")))
```
