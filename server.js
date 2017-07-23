
require('dotenv').config();

const micro = require('micro');
const schedule = require('node-schedule');
const twitter = require('./lib/twitter');
const sekkis = require('./sekki.json');

const parseStartDate = str => {
  const [month, day] = str.split('-');
  return { month, day };
}

const toCronFormat = ({ month, day }) => `5 20 ${day} ${month} *`;

const instructions = sekkis.map(sekki => ({
  id: sekki.id,
  crontab: toCronFormat(parseStartDate(sekki.startDate)),
  tweet: `${sekki.title}. ${sekki.description} ${sekki.emoji}`,
}));

const getPostTweetJob = tweet => () => twitter.post(tweet);

const jobs = instructions.map(instruction => schedule.scheduleJob(
  instruction.crontab,
  getPostTweetJob(instructions.tweet)
));

// Keep alive...
module.exports = () => `
  <html lang="en">
    <body>
      <p>Go follow <a href="https://twitter.com/smallseasonsbot">@smallseasonsbot</a>!  </p>
    </body>
  </html>
`;
