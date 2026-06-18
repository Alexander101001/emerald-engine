import simpleGit from 'simple-git';

const git = simpleGit();

export async function autoDeploy() {
  try {
    const status = await git.status();
    if (status.files.length === 0) {
      console.log('deployer: no changes to deploy');
      return false;
    }
    await git.add('.');
    await git.commit('Emerald: autonomous update cycle');
    await git.push('origin', 'main');
    console.log('deployer: pushed to origin/main');
    return true;
  } catch (e) {
    console.error('deployer: failed —', e.message);
    return false;
  }
}
