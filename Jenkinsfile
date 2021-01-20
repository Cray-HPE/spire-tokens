@Library('dst-shared@master') _

dockerBuildPipeline {
    repository = "cray"
    imagePrefix = "cray"
    app = "spire-tokens"
    name = "spire-tokens"
    description = "Service for issuing spire join tokens"
        product = "csm"
    githubPushRepo = "Cray-HPE/spire-tokens
    githubPushBranches: /(release\/.*|master)/
}
