# machtiani

**Machtiani (mct)** is an experimental terminal based, locally ran code chat service. It's designed to work with real projects. Thousands of commits. Thousands of files. For most cases, you’ll get faster, higher quality answers at a fraction of the cost than token-guzzling alternatives. If you run a local model on your laptop, you won't find anything better.

So why vibe when you can fly?

- Efficiently finds right answer based on precise context out of thousands of files in a project.
- Choose any API provider that adheres to OpenAI API spec, including locally ran ollama or mxl-lm.
- Stands on own, but composes in the terminal with other command line tools, such Codex.
- <div style="display: flex; align-items: center; text-align: left;">
    <span style="font-size: 1.1em; margin-right: 0.5em;">
      Enjoy this demo video and show support  👉
    </span>
    <a href="https://givebutter.com/mnGQAj/machtiani" style="display: inline-block;">
      <img src="images/givebutter.svg" alt="Support machtiani" width="100" />
    </a>
  </div>

## mct <img src="images/heart-svgrepo-com.svg" alt="heart" width="20" style="vertical-align:middle; margin:0 0.2em;"/> codex

**Combine the strongest strengths with `--mode answer-only`:**

- `mct` excels at understanding large, real codebases — grabbing the _right_ context and returning exactly what needs to be changed, with file paths and functions — at minimal token cost.
- `codex` is a superb executor for code implementation, refactor, and test runs, but works best when given precise instructions and context.

### Example: Compose for Peak Productivity

```bash
# Use mct to derive a minimal, context-aware plan and feed it directly to codex:
codex "$(mct \"Add error handling for API calls\" --mode answer-only --model Qwen2.5-Coder-7B-Instruct-Q6)"
```

- **Mix match models:** have `mct` do the heavy lifting with any model of your choice, while codex uses o4-mini (or whatever you want) to implement.
- **Lower token usage.** `mct`'s highly targeted planning leverages the code/project structure, so LLMs do less redundant thought and token looping.
- **Win/Win:**
  - *Best case*: `codex` can implement the change directly, with step-by-step details and file paths distilled by `mct` — bypassing expensive, imprecise code search.
  - *Worst case*: `codex` still benefits by having all ambiguity boiled out, so it simply fills in implementation details.
- **Result:** Faster, more precise, and _cheaper_ answers for big projects. When you combine their strengths, you waste less time, fewer tokens — and get more reliable automation at the CLI.

## Quick Launch

1. Clone this project.

   ```bash
   git clone --recurse-submodules https://github.com/tursomari/machtiani
   ```

2. Install `mct`

   ```bash
   cd mct
   go install \
     -ldflags="$(go run ./generate_ldflags)" \
     ./cmd/mct
   cd -
   ```

   If you're curious, `-ldflags` runs `mct/generate_ldflags/main.go` to set the version (git OID) so that `mct` can let you know if you're using an incompatible version of the local chat service.


   If `mct` command isn't found, run this and/or add it as a line in your `~/.bashrc`, `~/.zshrc` or whatever you use for a shell.

   ```
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

3. Launch the local chat service.

   Make sure n machtiani project root directory where `docker-compose.yml` resides.

   ```bash
   docker-compose up --build --remove-orphans
   ```

4. Set your OpenAI api key and base url.

   ```
   export MCT_MODEL_BASE_URL="https://api.openai.com/v1"
   export MCT_MODEL_API_KEY=sk...
   ```

   If you're using another API provider, for example OpenRouter.

   ```
   export MCT_MODEL_BASE_URL="https://openrouter.ai/api/v1"
   export MCT_MODEL_API_KEY=sk...
   ```

   If the git remote url (e.g. on Github) of the project you intend to use `mct` with is private (i.e., requires password or api key).

   ```
   export CODE_HOST_API_KEY=ghp...
   ```

   This works for any codehost, so it's not locked into Github.

5. Put a project on machtiani.

   In a git project you want to work on, run

   ```bash
   mct sync --model google/gemini-2.0-flash-001 --model-threads 10
   ```

   Give it time to finish. Run `mct status` to check back if completed.

6. Chat with the project


   ```bash
   mct "Ask whatever you want here" --model anthropic/claude-3.7-sonnet:thinking
   ```

7. Sync any new commits you pushed to your remote `origin`.

   ```bash
   mct sync --model gpt-4o-mini --model-threads 10
   ```

   Any local git commits must be pushed to the git remote for `mct` to sync it.


 No 'yolo'. mct agentic ability is constrained to:

 - applying git patches

 - reading files checked into git

 - saving chat convos and patch history to `.machtiani/`

## Configuration

Machtiani can be driven either by environment variables **or** by a YML machtiani-config file. Environment variables always win if set, but if you prefer to keep your keys and URLs in a file, you can place a `.machtiani-config.yml` in one of two locations:

1. **Project-local**
   `<your‐repo‐root>/.machtiani-config.yml`
2. **Home directory**
   `~/.machtiani-config.yml`

Lookup and precedence:

1. If you set an env var (for example `MCT_MODEL_API_KEY`), it always overrides any value from a config file.
2. Otherwise mct will look for **project-local** `./.machtiani-config.yml` first.
3. Failing that, it will fall back to your **home** `~/.machtiani-config.yml`.
4. If neither exists you must use env vars.

Supported keys in the yaml are exactly the same names as the env vars:

Example of a project-local file (`./.machtiani-config.yml`):

```yml
environment:
  MCT_MODEL_BASE_URL: "https://api.openai.com/v1"
  MCT_MODEL_API_KEY: "sk-REPLACE_WITH_YOURS"
  CODE_HOST_API_KEY: "ghp-REPLACE_WITH_YOURS"
}
```

You can omit any key that you still prefer to supply via `export`. Any missing key will simply fall back to its corresponding environment variable (or error out if not set anywhere).

## How it Works

Let's sync the commits a project. You must do this whether you have one initial commit or 1,000 commits in history (or more). It will make some inference requests, then **locally** generate and save some embeddings (cpu + 8GB RAM is just fine).

First, navigate to the directory where the git project resides.

```bash
mct sync --model gpt-4o-mini --model-threads 10
```

Initial sync of a fresh project to mct of a few hundred commits will take a couple minutes and cost few pennies (default is gpt-4o-mini). But once it's done it's done. It saves the data locally.

***Time and cost is practically nothing on incremental syncs on new commits.*** You can sync as many commits as needed (thousands upon thousands).

> **Speed Up Your Syncs**: The sync process is only constrained by how many requests per second you can make to your LLM provider. By increasing `--model-threads`, you can dramatically speed up syncing. For example, OpenRouter allows 1 request per second for every credit available on your account. With adequate credits and a capable system, using `--model-threads 100` could sync thousands of commits in seconds rather than minutes.


```bash
mct [prompt] --model gpt-4o-mini
```

It will apply code changes, if applicable. If it's not in git, **it doesn't exist**. Just remember that. So if you want to agentically run tests, or some other follow up workflow, use Codex. Codex can also more easily investigate '90% there' or run tests on a slam-dunk by mct, then trying itself from scratch (expensive and gets confused when it gets more complicated).

Now you can choose any model. If you have OpenRouter, you can use any openrouter model. Otherwise, plug in whatever your provider offers and it complies with OpenAI API format.

```bash
mct [prompt] --model deepseek/deepseek-r1
```

Also, you can choose for it not to edit or create new files based on the conversation.

```bash
mct [prompt] --mode chat --model gpt-4o-mini
```

## FYI

If we do `mct sync` with `amplify` flag, it will drastically increase the accuracy. This is a bail out if your commit history is poor and not well organized, but it costs more. Or if you want to guaranteee absolute peak performance.

```bash
mct sync --amplify low --model gpt-4o-mini --model-threads 10
```

`sync --amplify low` is about 2 times more costly and somewhat slower.

For example, say we have 15,000 commits. With low amplification, that would be about $0.50 with gpt-4o-mini (default).

But we could instead

```bash
mct sync --amplify high --depth 5000 --model gpt-4o-mini --model-threads 10
```

And that would only sync the most 5000 commits.

But let's scratch that for a sec and say we want all the commits available to machtiani. We could sync 9,999 of the oldest to newest.

```bash
git checkout HEAD~5000
mct sync --model gpt-4o-mini --model-threads 10
```

Then to sync the most recent 5001 commits

```bash
git checkout master
mct sync --amplify low --model gpt-4o-mini --model-threads 10
```

That way we have full coverage.

## Peak Performance

`sync --amplify high` is about **20 times more costly and 5 times slower**, compared to only **2 times** the cost and somewhat slower with `low`.

So it's always a good option for incremental syncs and will make sure you have peak performance going forward, or if you're not terribly cost sensitive for initial syncs.

## mct CLI Usage

### Overview

The machtiani cli `mct` allows you to interact with the project through command-line parameters. You can provide a markdown file or a prompt directly via the command line, along with various options such as the project name, model type, match strength, and mode of operation.

### Command Structure

```bash
mct [prompt] [flags]
```

### Flags

Common flags when chatting with a project:

- `--file <path>`            Use a markdown file as the conversation prompt.
- `--model <string>`         LLM model name (e.g. gpt-4o-mini, deepseek/deepseek-r1). Default: `gpt-4o-mini`.
- `--match-strength <level>` Context match strength (`high`, `mid`, `low`). Default: `mid`.
- `--mode <mode>`            Retrieval mode (`chat`, `pure-chat`, `answer-only`). Default: `commit`.
- `--force`                  Skip confirmation prompts.
- `--verbose`                Enable verbose logging.

Sync (`mct sync`) and remove (`mct remove`) commands support additional flags such as `--model-threads`, `--amplify`, `--depth`, `--cost-only`, etc. For the
full list of flags and detailed usage, run:

```bash
mct help
```

### Example Usage

1. **Providing a direct prompt, without apply git patches:**

   ```bash
   mct "Add a new endpoint to get stats." --model gpt-4o-mini --mode chat
   ```

2. **Using an existing markdown chat file:**
   ```bash
   mct --file .machtiani/chat/add_state_endpoint.md --model gpt-4o-mini
   ```

3. **Make it more selective to reduce context size:**

   ```bash
   mct "Add a new endpoint to get stats." --model gpt-4o --match-strength high
   ```

4. **Using the `--force` flag to skip confirmation:**

   ```bash
   mct sync --force --model gpt-4o-mini --model-threads 10
   ```

### Different Modes

In default `commit` mode, it searches commits for possible files to help answer the prompt. In `pure-chat` mode, it does not retrieve any files. In `chat` mode, it doesn't apply implement changes with git patches.

#### `sync`

The `sync` command is used to sync machtiani with the project's git..

**Usage:**
```bash
mct sync [--force] [--cost-only] [--model MODEL] [--model-threads NUM] [--amplify LEVEL] [--depth NUM]
```

**Parameters:**
- `--model` (optional): Specify which LLM to use (e.g., `gpt-4o-mini`, `gpt-4o`). Default is `gpt-4o-mini`.
- `--model-threads` (optional): Number of concurrent LLM requests to make during sync. Higher values mean faster syncing but require more API throughput and system resources.
- `--amplify` (optional): Amplification level (`off`, `low`, `mid`, `high`). Higher levels improve accuracy but increase cost.
- `--depth` (optional): Number of commits to sync, starting from the most recent.
- `--force` (optional): Skip confirmation prompts.
- `--cost-only` (optional): Estimate token usage without performing the actual sync.

**Example:**
```bash
mct sync --model gpt-4o-mini --model-threads 10 --amplify low --force
```

If you just want to estimate how many tokens it will require to sync:

```bash
mct sync --cost-only --model gpt-4o-mini
```

### `remove`

The `remove` command allows you to remove a repository from the Machtiani system.

**Usage:**
```bash
mct remove [--force]
```

**Example:**
```bash
mct remove
```

### Ignoring Files with `.machtiani.ignore`

You can ignore any binary files by providing the full path, such as images, etc. To exclude specific files from being processed by the application, you can create a `.machtiani.ignore` file in the root of your project directory. The files listed in this file will be ignored during the retrieval process.

#### Example `.machtiani.ignore` file:
```
poetry.lock
go.sum
go.mod
```

### Output

The CLI will stream the response and save the chat in `.machtiani/chat/` in the directory you ran the prompt. It also gives a descriptive name to the chat file for you convenience

## Developer Section

### End-to-End Tests

This project includes several end-to-end tests that validate the functionality of the Machtiani commands, with `test_end_to_end.py` serving as the **defacto test** for the application.

#### Setup

1. Make sure you have `git lfs` installed.

   You can use your systems package manager or whatever is best for you.

   Make sure to run after doing the above.

   ```bash
   git lfs install
   ```

   Make sure to run after doing the above.

   `git lfs install`

2. **Create and activate a Python virtual environment in the project root:**

   ```bash
   python3 -m venv venv  # Create a virtual environment named 'venv'
   source venv/bin/activate  # Activate the virtual environment (Linux/macOS)
   ```

3. **Install test dependencies using Poetry:**

   Navigate to the project root dir and run:

   ```bash
   poetry install
   ```

4. Install `all-MiniLM-L6-v2` into `end-to-end-tests/data/`

   ```
   cd end-to-end-tests/data
   git clone https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2
   ```
   It's a distilled Sentence-BERT embedding model. And it will work just fine on the cpus of a laptop.

   This is used to help test whether generated git messages by machtiani are in the ballpark of being correct using a cosine similarity threshold.

5. Clone `chastler` and `machtiani` into `end-to-end-tests/data/git-projects`

   ```
   cd end-to-end-tests/data/git-projects
   git clone https://github.com/7db9a/chastler
   git clone --branch end-to-end-test --single-branch https://github.com/7db9a/machtiani-end-to-end-test
   ```

   Unless you have the codheost keys for the repo, this may not work.

#### Running the Defacto Tests

To run the end-to-end test suite from inside `end-to-end-tests` directory:

```bash
python -m unittest test_end_to_end test_end_to_end_extra
```

`test_end_to_end_no_codehost_api_key` is no longer useful as count tokens requires auto-deletion of the repo on dry-run on initialization.

#### Why `test_end_to_end.py` is the Defacto Test

- **Comprehensive Coverage**: This test suite encompasses multiple critical functionalities of the Machtiani CLI, including the `git-store`, prompt handling, and synchronization with remote repositories. By running this test, you verify the overall integration and behavior of the CLI tool in a realistic scenario.

- **Realistic Environment**: The tests are designed to execute in a realistic environment, mimicking actual user interactions with the CLI. This helps in identifying issues that may not be apparent in isolated unit tests.

- **Validation of Core Features**: As it encompasses key functionalities, running this test ensures that the essential features of Machtiani are working as expected.

This command prioritizes the most critical integration tests, ensuring that your core functionalities more cost effectively (only a single round of setup and teardown).

### Other Tests

In addition to `test_end_to_end.py`, there are other tests available, such a below. However, it is recommended to prioritize the defacto tests above for a more focused validation of the core features. There no guarantee that the other tests will be maintained or its documentation kept up-to-date.

1. **Test for `git-store`**

   - **File**: `end-to-end-tests/test_git_store.py`
   - **Description**: Verifies the `git-store` command functionality.

2. **Test for Prompt Command**

   - **File**: `end-to-end-tests/test_prompt_command.py`
   - **Description**: Validates the behavior of the prompt command when querying specific questions.

To run all tests, you can still use:

```bash
python -m unittest discover .
```
