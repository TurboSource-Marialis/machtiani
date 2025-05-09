from fastapi import APIRouter, HTTPException
import git

router = APIRouter()

message = (
    "If building from source, run `./build.sh` in `machtiani/`.\n\n"
    "For installer, run:\n"
    "    curl -L https://raw.githubusercontent.com/tursomari/machtiani-releases/main/install.sh | bash\n"
    "     or\n"
    "    wget -O - https://raw.githubusercontent.com/tursomari/machtiani-releases/main/install.sh | bash"
)


@router.get("/get-head-oid")
async def get_head_oid():
    try:
        # Open the current repository
        repo = git.Repo(search_parent_directories=True)
        # Get the HEAD commit's OID
        head_oid = repo.head.commit.hexsha
        return {
                "head_oid": head_oid,
                "message": message
        }
    except git.exc.InvalidGitRepositoryError:
        raise HTTPException(status_code=404, detail="Not a git repository")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error retrieving HEAD OID: {str(e)}")
