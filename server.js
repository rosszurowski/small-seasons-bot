
require('dotenv').config();

const schedule = require('node-schedule');
const twitter = require('./lib/twitter');
const sekkis = require('./sekki.json');

const parseStartDate = (str) => {
  const [month, day] = str.split('-').map(n => parseInt(n, 10));
  return { month, day };
};

const toCronFormat = ({ month, day }) => `02 16 ${day} ${month} *`;

const instructions = sekkis.map(sekki => ({
  id: sekki.id,
  crontab: toCronFormat(parseStartDate(sekki.startDate)),
  tweet: `${sekki.title}. ${sekki.description} ${sekki.emoji}`,
}));

const getPostTweetJob = tweet => () =>
  twitter.post(tweet)
    .then(() => console.log('Posted tweet'))
    .catch(err => console.error(err));

instructions.map(instruction => schedule.scheduleJob(
  instruction.crontab,
  getPostTweetJob(instruction.tweet),
));

// Keep alive...
module.exports = () => `
  <html lang="en">
    <body>
      <p>Go follow <a href="https://twitter.com/smallseasonsbot">@smallseasonsbot</a>!</p>
    </body>
  </html>
`;
