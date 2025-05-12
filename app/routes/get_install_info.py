from fastapi import APIRouter, HTTPException
import git

router = APIRouter()

message = (
    'Rebuild your mct cli:\n\n'
    'See README.md or run the below command in your terminal in the machtiani project directory:\n\n'
    ' ==================================\n'
    '  cd mct\n'
    '  go install \\\n'
    '    -ldflags="$(go run ./generate_ldflags)" \\\n'
    '    ./cmd/mct\n'
    '  cd -\n'
    ' =================================='
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
