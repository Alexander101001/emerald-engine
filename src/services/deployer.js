import simpleGit from 'simple-git';

const git = simpleGit();

export async function pushToCloud(improvementCode) {
  console.log("Writing improvement to files...");
  await git.add('.');
  await git.commit('Self-evolution: Automated improvement');
  await git.push('origin', 'main');
  console.log("Pushed to GitHub/Hugging Face successfully.");
}
