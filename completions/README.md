# Gherkio Editor Completions

This directory contains auto-generated completion files for various text editors. These files are automatically updated via a GitHub Action whenever the Gherkin steps in the project are changed.

## Visual Studio Code

VS Code natively supports project-level snippets.

1.  **Create `.vscode` Directory**: If it doesn't already exist, create a `.vscode` directory in the root of your project.

2.  **Copy the Snippets File**: Copy the `gherkio.code-snippets` file into the `.vscode` directory. You can do this from your terminal:
    ```bash
    cp completions/gherkio.code-snippets .vscode/
    ```

3.  **Start Typing**: That's it! When you open a `.feature` file, VS Code will automatically suggest the Gherkin steps as you type.

## Vim / Neovim

You can configure Vim/Neovim to use the `gherkio.vim` file for autocompletion. The method depends on your setup.

### Method 1: Using Vim's Built-in Completion

This method uses Vim's native completion functionality without requiring any plugins.

1.  **Create Filetype Plugin Directory**: Create a filetype-specific plugin file for Gherkin (which typically has the `cucumber` filetype).
    *   For Vim: `mkdir -p ~/.vim/after/ftplugin/`
    *   For Neovim: `mkdir -p ~/.config/nvim/after/ftplugin/`

2.  **Create the Plugin File**: Create a file named `cucumber.vim` inside that directory and add the following line.

    **Important**: Replace `/path/to/your/project/` with the absolute path to this project's directory.
    ```vim
    " Add the gherkio completion file to the 'complete' option
    setlocal complete+=k/path/to/your/project/completions/gherkio.vim
    ```

3.  **Usage**: When editing a `.feature` file in Insert mode, press `Ctrl-X Ctrl-L` to trigger line completion. Vim will suggest steps from the generated file.

### Method 2: Using a Completion Plugin (e.g., coc.nvim)

If you use a modern completion plugin like `coc.nvim`, you can add the file as a dictionary source.

1.  **Edit CoC Configuration**: Open your CoC settings file by running `:CocConfig` in Vim/Neovim.

2.  **Add Dictionary Source**: Add or modify the following properties in your `coc-settings.json`. Remember to replace `/path/to/your/project/` with the absolute path.
    ```json
    {
      "coc.source.dictionary.filetypes": ["cucumber", "gherkin"],
      "coc.source.dictionary.words": {
        "cucumber": ["/path/to/your/project/completions/gherkio.vim"]
      }
    }
    ```
3.  **Usage**: CoC will now automatically suggest the Gherkin steps as you type in `.feature` files.
