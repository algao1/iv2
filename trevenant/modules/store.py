import gridfs

from datetime import datetime
from pymongo import MongoClient


class Store:
    def __init__(self, uri: str) -> None:
        self.client = MongoClient(uri, serverSelectionTimeoutMS=3000)
        self.client.server_info()  # Ensure connection is valid.
        self.database = self.client["ichor"]

    def get_glucose(self, start: datetime, end: datetime) -> list:
        return self.get_event("glucose", start, end)

    def get_insulin(self, start: datetime, end: datetime) -> list:
        return self.get_event("insulin", start, end)

    def get_carbs(self, start: datetime, end: datetime) -> list:
        return self.get_event("carbs", start, end)

    def get_event(self, event: str, start: datetime, end: datetime) -> list:
        col = self.database[event]
        cursor = col.find({"time": {"$gte": start, "$lt": end}}).sort("time", 1)
        return list(cursor)

    def store_image(self, contents, filename: str):
        database = self.client["ichor"]
        fs = gridfs.GridFS(database)
        file = fs.find_one({"filename": filename})
        if file:
            return file._id
        return fs.put(contents, filename=filename)

    def retrieve_image(self, iid: str, filename: str = ""):
        database = self.client["ichor"]
        fs = gridfs.GridFS(database)
        if not fs.exists(iid):
            raise FileNotFoundError(f"{iid} does not exist")
        return fs.get(iid)

