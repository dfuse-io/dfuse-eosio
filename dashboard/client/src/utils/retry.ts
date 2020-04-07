export const wait = (ms: number) => new Promise(r => setTimeout(r, ms));

export const retryFunc = async (func: Function, waitTime: number = 2000) => {
  let keepTrying;

  do {
    try {
      await func();
      console.log(`calling function - ${func.name} succeeded`);
      keepTrying = false;
    } catch (error) {
      console.log(
        `calling function - ${func.name} threw error: ${error}, keep retrying after ${waitTime}ms`
      );
      keepTrying = true;
    }
    await wait(waitTime);
  } while (keepTrying);
};
