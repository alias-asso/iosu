const infinite = document.querySelector("#infinite");
const datesField = document.querySelector("#contest-dates");
const dates = document.querySelectorAll("#contest-dates input");
const mode = document.querySelector("#mode");
const autoGenerate = document.querySelector("#auto-generate");
const autoGenerateField = document.querySelector("#auto-generate-field");

function updateContestDates() {
  datesField.hidden = infinite.checked;
  for (const input of dates) {
    input.disabled = infinite.checked;
    input.required = !infinite.checked;
  }
}

function updateAutoGenerate() {
  const freePlay = mode.value === "free_play";
  autoGenerateField.hidden = freePlay;
  autoGenerate.disabled = freePlay;
  if (freePlay) {
    autoGenerate.checked = false;
  }
}

infinite.addEventListener("change", updateContestDates);
mode.addEventListener("change", updateAutoGenerate);
updateContestDates();
updateAutoGenerate();
